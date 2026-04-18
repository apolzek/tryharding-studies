---
title: Quick guide to chart types and when to use them
tags: [observability, grafana, data-visualization, python]
status: stable
---

## Quick guide to chart types and when to use them

### Objectives

Build a quick reference mapping the most common chart types to the questions they answer best. The goal is not statistical depth but a fast decision aid when picking a visualization for a dashboard, report, or analysis — and a small runnable example for each chart type using Python and matplotlib.

### Prerequisites

- Python 3.9+
- Dependencies from `requirements.txt` (`pip install -r requirements.txt`)

### Reproducing

```sh
pip install -r requirements.txt

python bar.py
python linha.py
python pizza.py
python dispercao.py
python distribuicao.py
python mapa.py
```

Each script renders a minimal example of one chart type so you can tweak data and styling to build intuition.

### Results

| Chart Type | Description | When to use |
|---|---|---|
| **Bar Chart** | Vertical or horizontal bars to compare categories | Compare discrete values across groups (sales by product, population by state) |
| **Line Chart** | Connects points to show trends over time | Track evolution or temporal trends (monthly revenue, daily temperature) |
| **Pie Chart** | Circle divided into proportional slices | Show proportions in small sets (≤5 categories) |
| **Histogram** | Frequency distribution of continuous data in intervals | Visualize distributions (age, scores, heights) |
| **Scatter Plot** | Points showing relationships between two numeric variables | Examine correlations, patterns, clusters |
| **Area Chart** | Line chart with the area below filled | Emphasize volume or cumulative values over time |
| **Boxplot** | Median, quartiles, and outliers summary | Compare dispersion and outliers across groups |
| **Heatmap** | Color-coded matrix values | Density, correlation, or usage matrices |
| **Donut Chart** | Pie chart with a hole in the middle | Parts-of-a-whole with extra info in the center |
| **Bubble Chart** | Scatter plot where size encodes a third variable | Three-variable relationships |
| **Radar Chart** | Multiple variables in a radial layout | Compare multi-dimensional profiles |

Tips to master charts:

- Choose the right chart for the question
- Less is more: avoid overloading with information
- Simple charts for general audiences, detailed ones for experts
- Use color purposefully — highlight what matters
- Sometimes an unexpected chart tells the story better

### References

```
🔗 https://matplotlib.org/stable/gallery/index.html
🔗 https://www.data-to-viz.com/
🔗 https://datavizcatalogue.com/
```
