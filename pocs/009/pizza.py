import plotly.express as px
import pandas as pd

# Gerando dados fictícios
dados = pd.DataFrame({
    'Categoria': ['Eletrônicos', 'Roupas', 'Alimentos', 'Móveis', 'Brinquedos', 'Livros'],
    'Vendas': [45000, 28000, 52000, 22000, 15000, 17000]
})

# Criando o gráfico de pizza
fig = px.pie(
    dados,
    names='Categoria',
    values='Vendas',
    title='Distribuição de Vendas por Categoria de Produto',
    color_discrete_sequence=px.colors.qualitative.Set3,  # Cores mais suaves
    hole=0.3  # Define como gráfico de rosca; coloque 0 para pizza comum
)

# Personalizando layout
fig.update_traces(textinfo='percent+label')  # Mostra percentual e nome
fig.update_layout(
    width=600,
    height=600,
    showlegend=True,
    paper_bgcolor='white'
)

# Exibindo o gráfico
fig.show()
