import { NavLink, Route, Routes } from 'react-router-dom';
import Home from './pages/Home.jsx';
import Rights from './pages/Rights.jsx';
import Taxes from './pages/Taxes.jsx';
import Politicians from './pages/Politicians.jsx';
import History from './pages/History.jsx';

const nav = [
  { to: '/', label: 'Início', end: true },
  { to: '/direitos', label: 'Direitos & Normas' },
  { to: '/impostos', label: 'Impostos' },
  { to: '/politicos', label: 'Políticos' },
  { to: '/historia', label: 'História' },
];

export default function App() {
  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <span className="flag" aria-hidden>
            <span className="flag-green" />
            <span className="flag-diamond" />
            <span className="flag-circle" />
          </span>
          <div>
            <h1>Brasilzão</h1>
            <small>Tudo que um brasileiro precisa saber</small>
          </div>
        </div>
        <nav className="main-nav">
          {nav.map((n) => (
            <NavLink key={n.to} to={n.to} end={n.end} className={({ isActive }) => (isActive ? 'active' : '')}>
              {n.label}
            </NavLink>
          ))}
        </nav>
      </header>

      <main className="content">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/direitos" element={<Rights />} />
          <Route path="/direitos/:id" element={<Rights />} />
          <Route path="/impostos" element={<Taxes />} />
          <Route path="/politicos" element={<Politicians />} />
          <Route path="/politicos/:sigla" element={<Politicians />} />
          <Route path="/historia" element={<History />} />
        </Routes>
      </main>

      <footer className="footer">
        <small>
          Dados políticos via <a href="https://dadosabertos.camara.leg.br" target="_blank" rel="noreferrer">Câmara dos Deputados</a> e <a href="https://www12.senado.leg.br/dados-abertos" target="_blank" rel="noreferrer">Senado Federal</a>. Conteúdo educativo, não substitui consulta jurídica.
        </small>
      </footer>
    </div>
  );
}
