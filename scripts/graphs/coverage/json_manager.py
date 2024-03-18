import json
from pprint import pprint
import sys

# filePath = sys.argv[1]
# print(filePath)
# with open(filePath) as f:
#     data = json.load(f)

# print(len(data))
# for k,v in data.items():
#     print(k + " " + str(len(v)))

def load_json_file(filePath):
    with open(filePath) as f:
        data = json.load(f)
    return data
