import sys
from os import listdir
import os
import json_manager
import plot
import numpy as np

plotName = "PureExpl"
horizon = 50

f1 = []
f2 = ["Term", "random", "bonusRlMax", "neg"]
f3 = ["Diff", "random", "bonusRlMax", "neg"]
f4 = ["", "random", "bonusRlMax", "neg", "negVisits"]

# list of names to include (enough to be included in the name), if empty include all
filterNames = f3

# map of renaming rules for experiment names (key is the name to be replaced, value is the new name that will appear in the plot)
renamingMap = {
    "bonusRlMax": "BonusMaxRL",
    "negVisits": "NegRLVisits",
    "neg": "NegRL",
}

# target folder, it should contain all the target .json files
folder = sys.argv[1]
targetFolder = folder + "/ouputs"
if not os.path.exists(targetFolder):
    os.makedirs(targetFolder)


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
stdDevsSets = {}

# final result
finalCoverage = {}
finalCoverageAvg = {}
finalCoverageSD = {}

for expName, expData in dataSets.items():
    avgData = []
    stdDevs = []
    for i in range(shortestLen):
        x = []
        for data in expData:
            x.append(data[i])
        avgData.append(np.mean(x))
        stdDevs.append(np.std(x))
        if i == (shortestLen - 1):
            finalCoverageAvg[expName] = np.mean(x)
            finalCoverage[expName] = x
            finalCoverageSD[expName] = np.std(x)
    avgDataSets[expName] = avgData
    stdDevsSets[expName] = stdDevs


# create the plot
p = plot.multilinePlotShortestCustomSD(avgDataSets, stdDevsSets, 2, "", 0, horizon, "upper left")
p.savefig(targetFolder + "/" + plotName.replace(" ","_") + ".png", bbox_inches="tight")
p.savefig(targetFolder + "/" + plotName.replace(" ","_") + ".pdf", bbox_inches="tight")
p.close()

# save the final coverage data
final = {}
final["coverage"] = finalCoverage
final["coverageAvg"] = finalCoverageAvg
final["coverageSD"] = finalCoverageSD
json_manager.save_json_file(targetFolder + "/" + plotName.replace(" ","_") + "_final.json", final)


