import plotly.express as px
import pandas as pd
import numpy as np

# Gerando 200 idades aleatórias entre 18 e 70
np.random.seed(42)  # Para reprodução
idades = np.random.normal(loc=35, scale=10, size=200).astype(int)
idades = np.clip(idades, 18, 70)  # Garante que as idades fiquem no intervalo desejado

# Criando DataFrame
dados = pd.DataFrame({'Idade': idades})

# Criando o histograma
fig = px.histogram(
    dados,
    x='Idade',
    nbins=15,  # Número de faixas (bins)
    title='Distribuição de Idades dos Clientes',
    color_discrete_sequence=['teal']
)

# Personalizando o layout
fig.update_layout(
    xaxis_title='Idade',
    yaxis_title='Número de Clientes',
    plot_bgcolor='rgba(240,240,240,0.95)',
    paper_bgcolor='white',
    width=800,
    height=500
)

# Exibindo
fig.show()

# Salvando como HTML
# fig.write_html("grafico_histograma_idades_clientes.html")
fig.show()