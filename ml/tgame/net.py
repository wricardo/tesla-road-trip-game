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
    def __init__(self, channels: int = N_CHANNELS, width: int = 32):
        super().__init__()
        self.body = nn.Sequential(
            nn.Conv2d(channels, width, 3, padding=1),
            nn.ReLU(),
            nn.Conv2d(width, width * 2, 3, padding=1),
            nn.ReLU(),
            nn.Conv2d(width * 2, width * 2, 3, padding=1),
            nn.ReLU(),
        )
        self.head = nn.Conv2d(width * 2, N_ACTIONS, 1)

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
    )
    net.load_state_dict(ckpt["state_dict"])
    net.eval()
    return net, int(ckpt["size"])


@torch.no_grad()
def greedy_action(net: PolicyNet, env, size: int) -> int:
    from .features import encode

    x = torch.from_numpy(pad(encode(env), size)).float().unsqueeze(0)
    mask = torch.from_numpy(env.legal_mask()).unsqueeze(0)
    logits = masked_logits(net(x), mask)
    return int(logits.argmax(dim=1).item())
