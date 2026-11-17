"""Policy network + feature padding.

Fully convolutional by design: the same 3x3 kernels work on every map size and
the policy is translation-equivariant, so a 10x10 training map teaches the
10x10 corner of a 20x20 map. Padding a map to a common square with zeros keeps
batch shapes uniform; zeros read as impassable/uninformative.
"""
from __future__ import annotations

import numpy as np
import torch
import torch.nn as nn

from .features import N_CHANNELS

N_ACTIONS = 4


def pad(features: np.ndarray, size: int) -> np.ndarray:
    """(C,H,W) -> (C,size,size)."""
    c, h, w = features.shape
    out = np.zeros((c, size, size), dtype=features.dtype)
    out[:, :h, :w] = features
    return out


class PolicyNet(nn.Module):
    """Action logits per cell, read off the car cell.

    The default dilation stack reaches ~35 cells. Without that receptive field a
    3x3-only stack sees 7x7, which cannot follow a corridor to a park the encoder
    no longer points at (planes 6-7 are local-only by default).
    """

    def __init__(
        self,
        channels: int = N_CHANNELS,
        width: int = 32,
        dilations: tuple[int, ...] = (1, 2, 4, 8, 1, 1),
    ):
        super().__init__()
        self.dilations = tuple(dilations)
        layers: list[nn.Module] = []
        c = channels
        for i, d in enumerate(self.dilations):
            out = width if i == 0 else width * 2
            layers += [nn.Conv2d(c, out, 3, padding=d, dilation=d), nn.ReLU()]
            c = out
        self.body = nn.Sequential(*layers)
        self.head = nn.Conv2d(c, N_ACTIONS, 1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        # (B,C,S,S) -> (B,4). The value is read off the car cell: channel 5 is
        # a one-hot of the car position, so the readout is a learned selection.
        h = self.body(x)
        q = self.head(h)                 # (B,4,S,S)
        car = x[:, 5:6]                  # (B,1,S,S)
        return (q * car).sum(dim=(2, 3))


def masked_logits(logits: torch.Tensor, mask: torch.Tensor) -> torch.Tensor:
    """Illegal actions get -1e9. Walls are excluded from the mask: moving into
    one loses the game, so the policy should never consider it."""
    return logits.masked_fill(~mask, -1e9)


def save_policy(path, net: PolicyNet, size: int, extra: dict | None = None) -> None:
    torch.save(
        {
            "state_dict": net.state_dict(),
            "in_channels": net.body[0].in_channels,
            "width": net.body[0].out_channels,
            "dilations": net.dilations,
            "size": size,
            **(extra or {}),
        },
        path,
    )


def load_policy(path: str, device: str = "cpu") -> tuple[PolicyNet, int]:
    ckpt = torch.load(path, map_location=device)
    net = PolicyNet(
        channels=ckpt.get("in_channels", N_CHANNELS),
        width=ckpt.get("width", 32),
        # Checkpoints trained before the dilation stack are all 3x3, 3 layers.
        dilations=tuple(ckpt.get("dilations", (1, 1, 1))),
    )
    net.load_state_dict(ckpt["state_dict"])
    net.eval()
    return net, int(ckpt["size"])


class ValueNet(nn.Module):
    """V(s): probability-like estimate that this state is close to a win.

    No distance planes here (see features.HINT_PLANES), so the dilations give
    the receptive field the 3x3 stack of PolicyNet does not have: 1,2,4,8,1,1
    reaches ~33 cells, enough to see a 20x20 map end to end.
    """

    def __init__(self, channels: int = N_CHANNELS, width: int = 32,
                 dilations: tuple[int, ...] = (1, 2, 4, 8, 1, 1)):
        super().__init__()
        self.dilations = tuple(dilations)
        layers: list[nn.Module] = []
        c = channels
        for d in self.dilations:
            layers += [nn.Conv2d(c, width, 3, padding=d, dilation=d), nn.ReLU()]
            c = width
        self.body = nn.Sequential(*layers)
        self.head = nn.Conv2d(width, 1, 1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        # (B,C,S,S) -> (B,) logits, read off the car cell
        v = self.head(self.body(x))
        car = x[:, 5:6]
        return (v * car).sum(dim=(2, 3)).squeeze(1)


@torch.no_grad()
def value_batch(net: ValueNet, feats) -> np.ndarray:
    x = torch.from_numpy(np.asarray(feats)).float()
    return torch.sigmoid(net(x)).cpu().numpy()


def save_value(path, net: ValueNet, size: int, extra: dict | None = None) -> None:
    torch.save(
        {
            "state_dict": net.state_dict(),
            "in_channels": net.body[0].in_channels,
            "width": net.head.in_channels,
            "dilations": net.dilations,
            "size": size,
            **(extra or {}),
        },
        path,
    )


def load_value(path: str, device: str = "cpu") -> tuple[ValueNet, int]:
    ckpt = torch.load(path, map_location=device)
    net = ValueNet(
        channels=ckpt.get("in_channels", N_CHANNELS),
        width=ckpt["width"],
        dilations=tuple(ckpt.get("dilations", (1, 2, 4, 8, 1, 1))),
    )
    net.load_state_dict(ckpt["state_dict"])
    net.eval()
    return net, int(ckpt["size"])


@torch.no_grad()
def greedy_action(net: PolicyNet, env, size: int, hints: bool = False) -> int:
    from .features import encode

    x = torch.from_numpy(pad(encode(env, hints=hints), size)).float().unsqueeze(0)
    mask = torch.from_numpy(env.legal_mask()).unsqueeze(0)
    logits = masked_logits(net(x), mask)
    return int(logits.argmax(dim=1).item())


def load_any(path: str, device: str = "cpu"):
    """(kind, net, size, hints) for either checkpoint kind.

    The head width is the discriminator: policy nets emit 4 action logits, value
    nets 1. `hints` records what a policy was trained on; checkpoints predating
    the opt-in change were hinted, which is what they were."""
    ckpt = torch.load(path, map_location=device)
    if ckpt["state_dict"]["head.weight"].shape[0] == 1:
        net, size = load_value(path, device)
        return "value", net, size, False
    net, size = load_policy(path, device)
    return "policy", net, size, bool(ckpt.get("hints", True))
