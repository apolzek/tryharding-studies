import re
import pandas as pd

event_template = pd.read_csv("/opt/flink/data/BGL/BGL_templates.csv")
event_template['Regex'] = event_template['EventTemplate'].apply(
    lambda v: re.compile('^' + str(re.escape(v)).replace(r'\<\*\>', '.*') + '$')
)

node_no_dict = {"A": 10, "B": 11, "C": 12, "D": 13, "E": 14, "F": 15, 'd': 13}
for i in range(10):
    node_no_dict[str(i)] = i

channel_values = ['C', 'E', 'S', 'D', 'A']
component_values = ['KERNEL', 'LINKCARD', 'APP', 'MMCS', 'HARDWARE', 'DISCOVERY',
                    'CMCS', 'BGLMASTER', 'MONITOR', 'SERV_NET']
level_values = ['INFO', 'FATAL', 'WARNING', 'SEVERE', 'ERROR', 'FAILURE']

columns = [
    'rack', 'midplane', 'node_type', 'node_no', 'control_IO', 'jid', 'uid', 'type', 'eventId',
    'channel_A', 'channel_C', 'channel_D', 'channel_E', 'channel_S',
    'component_APP', 'component_BGLMASTER', 'component_CMCS', 'component_DISCOVERY',
    'component_HARDWARE', 'component_KERNEL', 'component_LINKCARD', 'component_MMCS',
    'component_MONITOR', 'component_SERV_NET',
    'level_ERROR', 'level_FAILURE', 'level_FATAL', 'level_INFO', 'level_SEVERE', 'level_WARNING'
]
