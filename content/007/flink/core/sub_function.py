import re
from constants import event_template


def node_extractor(node_text):
    pattern = re.compile(
        r'^(R(?P<rack>\d+))'
        r'(?:-M(?P<midplane>\w+))?'
        r'(?:-(?P<node_type>N|L)(?P<node_no>\w+))?'
        r'(?:-(?P<control_IO>[CI]))?'
        r'(?:\:J(?P<jid>\d+))?'
        r'(?:-U(?P<uid>\d+))?'
        r'(?:-(?P<channel>[A-Z]))?'
    )
    match = pattern.match(node_text)
    return match if match else {'rack': None, 'midplane': None, 'node_type': None, 'node_no': None,
                                'control_IO': None, 'jid': None, 'uid': None, 'channel': None}


def parse_log_line(line):
    if " RAS " in line:
        typee = "RAS"
        first, second = line.split(" RAS ")
    else:
        count = line.count(" NULL ")
        if count == 1:
            typee = "NULL"
            first, second = line.split(" NULL ")
        elif count == 3:
            typee = "NULL"
            first1, first2, first3, second = line.split(" NULL ")
            first = " NULL ".join([first1, first2, first3])
        else:
            return None

    first_pattern_missing_comp = re.compile(
        r'(?P<id>\d+)\s+'
        r'(?P<date>\d{4}\.\d{2}\.\d{2})\s+'
        r'(?P<code1>\S+)\s+'
        r'(?P<time>\d{4}-\d{2}-\d{2}-\d{2}\.\d{2}\.\d{2}\.\d+)\s+'
        r'(?P<code2>\S+)\s*'
    )
    second_pattern_missing_comp = re.compile(
        r'(?P<comp1>\S+)\s+'
        r'(?P<level>\S+)\s*'
        r'(?P<content>.*)'
    )
    first_pattern_missing_comp2 = re.compile(
        r'(?P<id>\d+)\s+'
        r'(?P<date>\d{4}\.\d{2}\.\d{2})\s+'
        r'(?P<code1>\S+)\s+'
        r'(?P<time>\d{4}-\d{2}-\d{2}-\d{2}\.\d{2}\.\d{2}\.\d+)\s*'
    )

    first_match = first_pattern_missing_comp.match(first)
    second_match = second_pattern_missing_comp.match(second)
    if first_match and second_match:
        id_, date, code1, time, code2 = first_match.groups()
        comp1, level, content = second_match.groups()
        if code1 == "-":
            code2 = "-"
        return [id_, date, code1, time, code2, typee, comp1, level, content]
    elif second_match:
        first_match2 = first_pattern_missing_comp2.match(first)
        id_, date, code1, time = first_match2.groups()
        comp1, level, content = second_match.groups()
        code2 = "-"
        return [id_, date, code1, time, code2, typee, comp1, level, content]
    return None


def match_pattern(log_line):
    for i in range(len(event_template)):
        if event_template['Regex'][i].match(log_line):
            return int(event_template['EventId'][i].split("E")[-1])
    return 0


def safe_int(val):
    try:
        return int(val)
    except:
        return -1
