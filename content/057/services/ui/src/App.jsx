import React, { useEffect, useRef, useState } from 'react';
import { api, getToken, setToken, subscribeEvents } from './api.js';

function Auth({ onLogin }) {
  const [email, setEmail] = useState('demo@057.test');
  const [password, setPassword] = useState('demo12345');
  const [msg, setMsg] = useState('');

  const doLogin = async () => {
    const r = await api.login(email, password);
    if (r.access_token) {
      setToken(r.access_token);
      setMsg('logged in');
      onLogin?.();
    } else setMsg(r.error || 'login failed');
  };
  const doRegister = async () => {
    const r = await api.register(email, password);
    setMsg(r.id ? `registered ${r.id} — now click login` : (r.error || 'register failed'));
  };

  return (
    <section>
      <h2>Auth <span className="pill">Go · JWT · Redis</span></h2>
      <input placeholder="email" value={email} onChange={(e) => setEmail(e.target.value)} />
      <input placeholder="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
      <button className="primary" onClick={doRegister}>register</button>
      <button className="primary" onClick={doLogin} style={{ marginLeft: '.4rem' }}>login</button>
      {getToken() && <pre>token: {getToken().slice(0, 64)}…</pre>}
      <p>{msg}</p>
    </section>
  );
}

function Products() {
  const [items, setItems] = useState([]);
  const [form, setForm] = useState({ sku: '', name: '', price: 0, stock: 0, catalog_id: 'default' });
  const reload = async () => setItems((await api.listProducts()).items || []);
  useEffect(() => { reload(); }, []);
  const create = async () => {
    await api.createProduct({ ...form, price: Number(form.price), stock: Number(form.stock), description: '' });
    await reload();
  };
  return (
    <section>
      <h2>Products <span className="pill">Rust · gRPC · Postgres</span></h2>
      <div>
        {['sku','name','price','stock','catalog_id'].map((k) => (
          <input key={k} placeholder={k} value={form[k]} onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
        ))}
        <button className="primary" onClick={create}>create</button>
      </div>
      <table>
        <thead><tr><th>SKU</th><th>Name</th><th>Price</th><th>Stock</th><th>ID</th></tr></thead>
        <tbody>
          {items.map((p) => (
            <tr key={p.id}>
              <td>{p.sku}</td><td>{p.name}</td>
              <td>${p.price?.toFixed ? p.price.toFixed(2) : p.price}</td>
              <td>{p.stock}</td>
              <td style={{ fontFamily: 'monospace', fontSize: '.75rem' }}>{p.id}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Cart() {
  const cartId = localStorage.getItem('057.cart') || (() => { const v = `cart-${crypto.randomUUID()}`; localStorage.setItem('057.cart', v); return v; })();
  const [cart, setCart] = useState({ items: [], total: 0, ttl_seconds: -1 });
  const [form, setForm] = useState({ product_id: '', sku: '', qty: 1, unit_price: 0 });
  const reload = async () => setCart(await api.cart.get(cartId));
  useEffect(() => { reload(); }, []);
  const add = async () => {
    await api.cart.add(cartId, { ...form, qty: Number(form.qty), unit_price: Number(form.unit_price) });
    await reload();
  };
  return (
    <section>
      <h2>Cart <span className="pill">Node · Redis TTL</span></h2>
      <p><small>cart id: <code>{cartId}</code> · ttl: {cart.ttl_seconds}s</small></p>
      <div>
        {['product_id','sku','qty','unit_price'].map((k) => (
          <input key={k} placeholder={k} value={form[k]} onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
        ))}
        <button className="primary" onClick={add}>add</button>
        <button className="primary" style={{ marginLeft: '.4rem' }} onClick={async () => { await api.cart.clear(cartId); await reload(); }}>clear</button>
      </div>
      <table>
        <thead><tr><th>Product</th><th>SKU</th><th>Qty</th><th>Unit $</th><th></th></tr></thead>
        <tbody>
          {(cart.items || []).map((it) => (
            <tr key={it.product_id}>
              <td>{it.product_id}</td><td>{it.sku}</td><td>{it.qty}</td><td>{it.unit_price}</td>
              <td><button onClick={async () => { await api.cart.remove(cartId, it.product_id); await reload(); }}>×</button></td>
            </tr>
          ))}
        </tbody>
      </table>
      <p><b>Total:</b> ${cart.total?.toFixed?.(2) ?? cart.total}</p>
    </section>
  );
}

function Orders() {
  const [customerId, setCustomerId] = useState('');
  const [orders, setOrders] = useState([]);
  const [history, setHistory] = useState([]);
  const reload = async () => setOrders(customerId ? (await api.listOrders(customerId)) : []);
  return (
    <section>
      <h2>Orders <span className="pill">Python · Postgres · State machine · Outbox · Event sourcing</span></h2>
      <input placeholder="customer id" value={customerId} onChange={(e) => setCustomerId(e.target.value)} />
      <button className="primary" onClick={reload}>load</button>
      <table>
        <thead><tr><th>ID</th><th>Status</th><th>Total</th><th>Updated</th><th></th></tr></thead>
        <tbody>
          {(orders || []).map((o) => (
            <tr key={o.id}>
              <td style={{ fontFamily: 'monospace', fontSize: '.75rem' }}>{o.id}</td>
              <td><span className="pill">{o.status}</span></td>
              <td>${o.total?.toFixed?.(2)}</td>
              <td>{o.updated_at}</td>
              <td>
                <button onClick={async () => setHistory(await api.orderHistory(o.id))}>history</button>
                <button onClick={async () => { await api.fulfill(o.id); await reload(); }} style={{ marginLeft: '.2rem' }}>fulfill</button>
                <button onClick={async () => { await api.cancel(o.id); await reload(); }} style={{ marginLeft: '.2rem' }}>cancel</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {history.length > 0 && (
        <>
          <h3>Event log (event sourcing)</h3>
          <pre>{JSON.stringify(history, null, 2)}</pre>
        </>
      )}
    </section>
  );
}

function Checkout() {
  const [form, setForm] = useState({ name: 'Ada', email: 'ada@lovelace.test', document: '0001',
                                     product_id: '', quantity: 1, plan: 'premium', amount: 29.9 });
  const [result, setResult] = useState(null);
  const submit = async () => setResult(await api.checkout({
    ...form, quantity: Number(form.quantity), amount: Number(form.amount),
  }, crypto.randomUUID()));
  return (
    <section>
      <h2>Checkout saga <span className="pill">Saga · Compensation · Idempotency</span></h2>
      <div>
        {Object.keys(form).map((k) => (
          <input key={k} placeholder={k} value={form[k]} onChange={(e) => setForm({ ...form, [k]: e.target.value })} />
        ))}
        <button className="primary" onClick={submit}>run saga</button>
      </div>
      {result && <pre>{JSON.stringify(result, null, 2)}</pre>}
    </section>
  );
}

function SignaturesDashboard() {
  const [data, setData] = useState([]);
  useEffect(() => { api.summary().then((r) => setData(Array.isArray(r) ? r : [])); }, []);
  return (
    <section>
      <h2>Signatures dashboard <span className="pill">Python · ClickHouse · CQRS</span></h2>
      <table>
        <thead><tr><th>Plan</th><th>Total</th><th>Revenue</th></tr></thead>
        <tbody>
          {data.map((r) => (
            <tr key={r.plan}><td>{r.plan}</td><td>{r.total}</td><td>${r.revenue?.toFixed?.(2)}</td></tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Live() {
  const [events, setEvents] = useState([]);
  const ref = useRef(null);
  useEffect(() => {
    const es = subscribeEvents((evt) => {
      setEvents((cur) => [{ t: new Date().toLocaleTimeString(), ...evt }, ...cur].slice(0, 100));
    });
    return () => es.close();
  }, []);
  return (
    <section>
      <h2>Live event feed <span className="pill">SSE · RabbitMQ bridge</span></h2>
      <p><small>Streams every message published to <code>customer.events</code>, <code>order.events</code> and <code>payment.events</code>.</small></p>
      <div ref={ref} style={{ maxHeight: 300, overflow: 'auto' }}>
        {events.map((e, i) => (
          <div key={i} style={{ fontFamily: 'monospace', fontSize: '.8rem', borderBottom: '1px solid #1f2937', padding: '.3rem 0' }}>
            <span className="pill">{e.t}</span>{' '}
            <span className="pill" style={{ background: '#6366f1' }}>{e.data?.routingKey || e.type}</span>{' '}
            <code>{JSON.stringify(e.data?.body ?? e.data)}</code>
          </div>
        ))}
      </div>
    </section>
  );
}

export default function App() {
  const [tab, setTab] = useState('auth');
  const tabs = {
    auth: <Auth onLogin={() => setTab('products')} />,
    products: <Products />,
    cart: <Cart />,
    checkout: <Checkout />,
    orders: <Orders />,
    dashboard: <SignaturesDashboard />,
    live: <Live />,
  };
  return (
    <>
      <header>
        <h1>057 · Distributed microservices</h1>
        <nav>
          {Object.keys(tabs).map((k) => (
            <button key={k} className={tab === k ? 'active' : ''} onClick={() => setTab(k)}>{k}</button>
          ))}
        </nav>
      </header>
      <main>{tabs[tab]}</main>
    </>
  );
}
