import plotly.express as px
import pandas as pd

# Carregando os dados do Gapminder (dataset built-in do Plotly)
data = px.data.gapminder()

# Criando o gráfico de dispersão animado
fig = px.scatter(
    data,
    x="gdpPercap",           # PIB per capita no eixo X
    y="lifeExp",            # Expectativa de vida no eixo Y
    animation_frame="year",  # Animação por ano
    animation_group="country", # Agrupamento por país
    size="pop",             # Tamanho dos pontos baseado na população
    color="continent",      # Cor baseada no continente
    hover_name="country",   # Nome do país ao passar o mouse
    log_x=True,            # Escala logarítmica no eixo X
    size_max=60,           # Tamanho máximo dos pontos
    range_x=[200, 60000],  # Intervalo do eixo X
    range_y=[20, 90],      # Intervalo do eixo Y
    title="Gráfico Animado: Expectativa de Vida vs PIB Per Capita",
    labels={
        "gdpPercap": "PIB Per Capita (USD)",
        "lifeExp": "Expectativa de Vida (anos)",
        "pop": "População"
    }
)

# Personalizando o layout
fig.update_layout(
    width=900,
    height=600,
    font=dict(size=12),
    showlegend=True,
    plot_bgcolor='rgba(240,240,240,0.95)',
    paper_bgcolor='white'
)

# Configurando a animação
fig.layout.updatemenus[0].buttons[0].args[1]['frame']['duration'] = 1000
fig.layout.updatemenus[0].buttons[0].args[1]['transition']['duration'] = 500

# Exibindo o gráfico
fig.show()

# Salvando o gráfico como arquivo HTML (opcional)
# fig.write_html("grafico_animado.html")

print("Gráfico criado com sucesso!")
print("\nInformações sobre o dataset:")
print(f"- Total de países: {data['country'].nunique()}")
print(f"- Período: {data['year'].min()} a {data['year'].max()}")
print(f"- Continentes: {', '.join(data['continent'].unique())}")