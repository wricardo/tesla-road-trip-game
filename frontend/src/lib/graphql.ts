import { createClient, cacheExchange, fetchExchange, type Client } from '@urql/svelte';
import { createClient as createWsClient, type Client as WsClient } from 'graphql-ws';
import { subscriptionExchange } from '@urql/svelte';

const GRAPHQL_URL = (typeof window !== 'undefined' && import.meta.env?.PUBLIC_GRAPHQL_URL)
  ? import.meta.env.PUBLIC_GRAPHQL_URL
  : '/graphql';

const WS_URL = (typeof window !== 'undefined' && import.meta.env?.PUBLIC_WS_URL)
  ? import.meta.env.PUBLIC_WS_URL
  : (typeof window !== 'undefined'
    ? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/graphql`
    : 'ws://localhost:8080/graphql');

let wsClient: WsClient | null = null;

function getWsClient(): WsClient {
  if (!wsClient) {
    wsClient = createWsClient({ url: WS_URL });
  }
  return wsClient;
}

export function makeClient(): Client {
  return createClient({
    url: GRAPHQL_URL,
    exchanges: [
      cacheExchange,
      fetchExchange,
      subscriptionExchange({
        forwardSubscription(request) {
          const input = { ...request, query: request.query ?? '' };
          return {
            subscribe(sink) {
              const unsubscribe = getWsClient().subscribe(input, sink);
              return { unsubscribe };
            }
          };
        }
      })
    ]
  });
}
