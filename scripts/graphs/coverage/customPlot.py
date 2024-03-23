import sys
from os import listdir
import json_manager

# list of names to include (enough to be included in the name), if empty include all
filterNames = []

# map of renaming rules for experiment names (key is the name to be replaced, value is the new name that will appear in the plot)
renamingMap = {}

# target folder, it should contain all the target .json files
folder = sys.argv[1]

jsonFiles = [f for f in listdir(folder) if f.endswith(".json")]

dataSets = {}

for file in jsonFiles:
    data = json_manager.load_json_file(folder + "/" + file)
