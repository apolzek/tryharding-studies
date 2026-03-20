import plotly.express as px
import pandas as pd

# Dados fictícios
dados = pd.DataFrame({
    'Região': ['Norte', 'Nordeste', 'Centro-Oeste', 'Sudeste', 'Sul'],
    'Faturamento': [48000, 39000, 27000, 86000, 52000]
})

# Criando o gráfico de barras
fig = px.bar(
    dados,
    x='Região',
    y='Faturamento',
    title='Faturamento por Região',
    text='Faturamento',
    color='Região',
    color_discrete_sequence=px.colors.qualitative.Vivid
)

# Personalizando o layout
fig.update_traces(texttemplate='R$ %{text:.2f}', textposition='outside')
fig.update_layout(
    uniformtext_minsize=8,
    uniformtext_mode='hide',
    yaxis_title='Faturamento (R$)',
    xaxis_title='Região',
    plot_bgcolor='rgba(240,240,240,0.95)',
    paper_bgcolor='white',
    width=800,
    height=500
)

# Exibindo o gráfico
fig.show()

# Salvando o gráfico como arquivo HTML
fig.write_html("grafico_barras_faturamento_regional.html")
