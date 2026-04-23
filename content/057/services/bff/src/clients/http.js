import axios from 'axios';
import CircuitBreaker from 'opossum';

/**
 * Wrap an Axios client with a circuit breaker.
 * Prevents cascading failures when a downstream service is slow/down:
 * after `errorThresholdPercentage` errors in a rolling window, the breaker
 * opens and fails fast for `resetTimeout` ms before half-opening.
 */
export function makeClient(baseURL, name) {
  const instance = axios.create({ baseURL, timeout: 3000 });

  const call = async ({ method = 'GET', url, data, headers }) => {
    const res = await instance.request({ method, url, data, headers });
    return res.data;
  };

  const breaker = new CircuitBreaker(call, {
    timeout: 3000,
    errorThresholdPercentage: 50,
    resetTimeout: 10_000,
    name,
  });

  breaker.fallback(() => ({ degraded: true, service: name }));

  return {
    request: (opts) => breaker.fire(opts),
    breaker,
  };
}
