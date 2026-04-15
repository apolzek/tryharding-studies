import { createApp } from 'vue';
import App from './App.vue';
import { initTelemetry } from './telemetry';
import './styles.css';

initTelemetry();

createApp(App).mount('#app');
