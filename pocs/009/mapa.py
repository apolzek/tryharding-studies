import plotly.express as px
import pandas as pd
import json
import urllib.request

# Dados fictícios
dados = pd.DataFrame({
    'Estado': ['SP', 'RJ', 'MG', 'ES', 'BA', 'RS', 'PR', 'SC', 'GO', 'MT'],
    'População': [46000000, 17000000, 21000000, 4000000, 15000000, 11500000, 11500000, 7500000, 7000000, 3500000]
})

# Baixar GeoJSON dos estados do Brasil (exemplo público)
url = 'https://raw.githubusercontent.com/codeforamerica/click_that_hood/master/public/data/brazil-states.geojson'
with urllib.request.urlopen(url) as response:
    brazil_states = json.load(response)

# No GeoJSON, cada estado tem a propriedade "sigla" com o código UF

fig = px.choropleth(
    dados,
    geojson=brazil_states,
    locations='Estado',
    color='População',
    color_continuous_scale='Viridis',
    range_color=(dados['População'].min(), dados['População'].max()),
    featureidkey='properties.sigla',  # chave no GeoJSON que corresponde a 'Estado'
    labels={'População':'População estimada'},
    title='População Estimada por Estado Brasileiro (Fictício)'
)

fig.update_geos(
    fitbounds="locations",  # ajusta o zoom para os estados mostrados
    visible=False
)

fig.update_layout(
    width=900,
    height=600
)

fig.show()
