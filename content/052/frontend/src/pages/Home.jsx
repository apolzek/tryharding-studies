import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api.js';

export default function Home() {
  const [tip, setTip] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.tipToday().then(setTip).catch((e) => setError(e.message));
  }, []);

  return (
    <div>
      <section className="hero">
        <span className="badge">Dica do dia</span>
        {error && <div className="error">Não foi possível carregar a dica: {error}</div>}
        {!tip && !error && <div className="loading">Carregando dica do dia…</div>}
        {tip && (
          <>
            <h2>{tip.category}</h2>
            <p>{tip.tip}</p>
            <span className="date">📅 {new Date(tip.date).toLocaleDateString('pt-BR', { weekday: 'long', day: '2-digit', month: 'long', year: 'numeric' })}</span>
          </>
        )}
      </section>

      <div className="card">
        <h3>O que você encontra aqui</h3>
        <p style={{ color: 'var(--muted)', marginTop: 0 }}>
          Conteúdo objetivo sobre direitos, impostos, política e a história do nosso país. Feito para o cidadão comum, com linguagem clara e fontes citadas.
        </p>
        <div className="grid">
          <Link to="/direitos" className="card category-card" style={{ margin: 0, borderTopColor: 'var(--verde)' }}>
            <h3>⚖️ Direitos & Normas</h3>
            <p>Consumidor, trânsito, trabalho, saúde, previdência, moradia, documentos, digital, penal.</p>
          </Link>
          <Link to="/impostos" className="card category-card" style={{ margin: 0, borderTopColor: 'var(--amarelo)' }}>
            <h3>💰 Impostos</h3>
            <p>IR, IPVA, IPTU, ICMS, ISS e todo o calendário tributário explicado.</p>
          </Link>
          <Link to="/politicos" className="card category-card" style={{ margin: 0, borderTopColor: 'var(--azul)' }}>
            <h3>🏛️ Políticos</h3>
            <p>Lista de deputados e senadores em exercício. Dashboards por partido com dados oficiais.</p>
          </Link>
          <Link to="/historia" className="card category-card" style={{ margin: 0, borderTopColor: 'var(--verde)' }}>
            <h3>📚 História do Brasil</h3>
            <p>Pré-colonial até a Nova República. Marcos, datas, contexto.</p>
          </Link>
        </div>
      </div>

      <div className="card">
        <h3>Atalhos oficiais úteis</h3>
        <div className="quick-links">
          <a href="https://www.gov.br" target="_blank" rel="noreferrer"><strong>gov.br</strong><span>Acesso único a serviços federais</span></a>
          <a href="https://consumidor.gov.br" target="_blank" rel="noreferrer"><strong>consumidor.gov.br</strong><span>Reclamações contra empresas</span></a>
          <a href="https://www.gov.br/receitafederal" target="_blank" rel="noreferrer"><strong>Receita Federal</strong><span>IR, CPF, situação fiscal</span></a>
          <a href="https://meu.inss.gov.br" target="_blank" rel="noreferrer"><strong>Meu INSS</strong><span>Benefícios e extrato previdenciário</span></a>
          <a href="https://registrato.bcb.gov.br" target="_blank" rel="noreferrer"><strong>Registrato (BC)</strong><span>Suas contas e dívidas bancárias</span></a>
          <a href="https://valoresareceber.bcb.gov.br" target="_blank" rel="noreferrer"><strong>Valores a Receber</strong><span>Dinheiro esquecido em bancos</span></a>
        </div>
      </div>
    </div>
  );
}
