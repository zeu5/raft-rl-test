import sys
from os import listdir
import json_manager
import plot

plotName = "LeaderTerm2"
horizon = 50

f1 = []
f2 = ["Term", "random", "bonusRlMax", "neg"]
f3 = ["Diff", "random", "bonusRlMax", "neg"]

# list of names to include (enough to be included in the name), if empty include all
filterNames = f2

# map of renaming rules for experiment names (key is the name to be replaced, value is the new name that will appear in the plot)
renamingMap = {
    "bonusRlMax": "BonusMaxRL",
    "negVisits": "NegRLVisits",
    "neg": "NegRL",
}

# target folder, it should contain all the target .json files
folder = sys.argv[1]

jsonFiles = [f for f in listdir(folder) if f.endswith(".json")]

dataSets = {}
shortestLen = 100000000000000

for file in jsonFiles:
    data = json_manager.load_json_file(folder + "/" + file)
    for expName, expData in data.items():

        contained = False
        for name in filterNames:
            if name in expName:
                contained = True
                break

        if len(filterNames) == 0 or contained:
            if expName in renamingMap:
                expName = renamingMap[expName]
            if expName not in dataSets:
                dataSets[expName] = []
            dataSets[expName].append(expData)
            if len(expData) < shortestLen:
                shortestLen = len(expData)

# calculate the average of the data
avgDataSets = {}

for expName, expData in dataSets.items():
    avgData = []
    for i in range(shortestLen):
        avg = 0
        for data in expData:
            avg += data[i]
        avgData.append(avg / len(expData))
    avgDataSets[expName] = avgData

# create the plot
p = plot.multilinePlotShortest(avgDataSets, "", 0, horizon, "upper left")
p.savefig(folder + "/" + plotName.replace(" ","_") + ".png", bbox_inches="tight")
p.savefig(folder + "/" + plotName.replace(" ","_") + ".pdf", bbox_inches="tight")
p.close()


