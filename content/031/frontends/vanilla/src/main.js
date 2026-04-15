import { initTelemetry } from './telemetry.js';
import { initFeed } from './feed.js';

initTelemetry();

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initFeed);
} else {
  initFeed();
}
