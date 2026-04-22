import { Router } from 'express';
import { listDeputados, listSenadores, aggregateByParty } from '../services/camara.js';

export const politiciansRouter = Router();

politiciansRouter.get('/deputados', async (_req, res, next) => {
  try {
    const list = await listDeputados();
    res.json({ total: list.length, deputados: list });
  } catch (e) { next(e); }
});

politiciansRouter.get('/senadores', async (_req, res, next) => {
  try {
    const list = await listSenadores();
    res.json({ total: list.length, senadores: list });
  } catch (e) { next(e); }
});

politiciansRouter.get('/partidos', async (req, res, next) => {
  try {
    const scope = (req.query.scope || 'all').toString();
    const [deputados, senadores] = await Promise.all([
      scope === 'senado' ? [] : listDeputados(),
      scope === 'camara' ? [] : listSenadores(),
    ]);
    const all = [...deputados, ...senadores];
    const byParty = aggregateByParty(all);
    const totals = {
      deputados: deputados.length,
      senadores: senadores.length,
      total: all.length,
      partidos: byParty.length,
    };
    res.json({ totals, partidos: byParty });
  } catch (e) { next(e); }
});

politiciansRouter.get('/partidos/:sigla', async (req, res, next) => {
  try {
    const [deputados, senadores] = await Promise.all([listDeputados(), listSenadores()]);
    const sigla = req.params.sigla.toUpperCase();
    const membrosCamara = deputados.filter((d) => d.partido === sigla);
    const membrosSenado = senadores.filter((s) => s.partido === sigla);
    if (!membrosCamara.length && !membrosSenado.length) {
      return res.status(404).json({ error: 'partido sem membros ou sigla inválida' });
    }
    const porUf = {};
    for (const m of [...membrosCamara, ...membrosSenado]) {
      porUf[m.uf] = (porUf[m.uf] || 0) + 1;
    }
    res.json({
      sigla,
      total: membrosCamara.length + membrosSenado.length,
      camara: membrosCamara.length,
      senado: membrosSenado.length,
      porUf,
      deputados: membrosCamara,
      senadores: membrosSenado,
    });
  } catch (e) { next(e); }
});
