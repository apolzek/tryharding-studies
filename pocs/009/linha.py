# linha.py

import plotly.express as px
import pandas as pd

# Contexto:
# Simulamos vendas mensais de uma loja online ao longo de um ano.

dados = pd.DataFrame({
    'Mês': ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'],
    'Vendas': [1200, 1500, 1800, 2100, 2500, 2900, 3200, 3500, 3700, 4100, 4600, 5000]
})

fig = px.line(
    dados,
    x='Mês',
    y='Vendas',
    title='Crescimento Mensal das Vendas Online em 2025',
    markers=True,
    line_shape='spline',
    labels={'Vendas': 'Número de Vendas'}
)

fig.update_layout(
    xaxis_title='Mês',
    yaxis_title='Vendas',
    plot_bgcolor='rgba(240,240,240,0.95)',
    paper_bgcolor='white',
    width=800,
    height=500
)

fig.show()
