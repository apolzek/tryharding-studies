import { useState } from 'react';
import { trace } from '@opentelemetry/api';
import Feed from './components/Feed';
import Stories from './components/Stories';
import Tabs, { type TabName } from './components/Tabs';
import { emitLog, getMeters, STACK_NAME } from './telemetry';

export default function App() {
  const [tab, setTab] = useState<TabName>('feed');

  const onChange = (next: TabName) => {
    if (next === tab) return;
    const tracer = trace.getTracer('tabs');
    const span = tracer.startSpan('tab.switch', {
      attributes: { 'tab.name': next, 'frontend.stack': STACK_NAME },
    });
    try {
      getMeters().tabSwitchesCounter?.add(1, {
        tab: next,
        'frontend.stack': STACK_NAME,
      });
    } catch {}
    const traceId = span.spanContext?.().traceId || '';
    emitLog({ event: 'tab.switch', tab: next, trace_id: traceId });
    span.end();
    setTab(next);
  };

  return (
    <>
      <header className="app-header">
        <h1>Attentium</h1>
        <span className="stack-badge">react</span>
      </header>
      <Tabs active={tab} onChange={onChange} />
      <main>
        {tab === 'feed' ? <Feed /> : <Stories />}
      </main>
    </>
  );
}
