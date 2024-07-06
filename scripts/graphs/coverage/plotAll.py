import sys
from os import listdir
import os
import json_manager
import plot
import numpy as np
import statTests

# call it on the outer folder of a set of experiments, after processing it with createDataFolders.py


horizon = 50

# map of renaming rules for experiment names (key is the name to be replaced, value is the new name that will appear in the plot)
renamingMap = {
    "bonusRlMax": "BonusMaxRL",
    "negVisits": "NegRLVisits",
    "neg": "NegRL",
}

# target folder, it should contain all the target .json files
folder = sys.argv[1]
folder = os.path.join(folder, "coverageData")

for expFolder in listdir(folder):
    if os.path.isdir(os.path.join(folder, expFolder)):
        expPath = os.path.join(folder, expFolder)
        targetFolder = os.path.join(folder, expFolder, "outputs")
        if not os.path.exists(targetFolder):
            os.makedirs(targetFolder)

        # read all the json files in the folder
        jsonFiles = [f for f in listdir(expPath) if f.endswith(".json")]

        dataSets = {} # set of policy names: list of data sets for multiple experiment runs
        shortestLen = 1000000000000000000

        for file in jsonFiles:
            data = json_manager.load_json_file(os.path.join(expPath, file))
            for expName, expData in data.items():

                if expName in renamingMap or expName == expFolder:
                    # eventually rename the policy (for the final plot)
                    if expName in renamingMap:
                        expName = renamingMap[expName]
                    else:
                        expName = "PredRL_" + expName
                    # if not existing, create a list of data sets for the policy
                    if expName not in dataSets:
                        dataSets[expName] = []
                    # add the data set to the list
                    dataSets[expName].append(expData)
                    if len(expData) < shortestLen: # keep track of the shortest data set
                        shortestLen = len(expData)

        # calculate the average of the data
        avgDataSets = {}
        stdDevsSets = {}

        # final result
        finalCoverage = {}
        finalCoverageAvg = {}
        finalCoverageSD = {}
        CoverageStatTests = {}

        for expName, expData in dataSets.items():
            avgData = []
            stdDevs = []
            for i in range(shortestLen): # foreach episode
                x = [] # collect the experiment data at the same episode for each experiment
                for data in expData:
                    x.append(data[i])
                avgData.append(np.mean(x)) # compute and append the average
                stdDevs.append(np.std(x)) # compute and append the standard deviation
                if i == (shortestLen - 1): # if last episode, compute and store the final values
                    finalCoverageAvg[expName] = np.mean(x)
                    finalCoverage[expName] = x
                    finalCoverageSD[expName] = np.std(x)
            avgDataSets[expName] = avgData
            stdDevsSets[expName] = stdDevs

        # calculate the statistical tests
        for expName, finalCovList in finalCoverage.items():
            if expName not in {"random", "BonusMaxRL", "NegRLVisits"}:
                if expName not in CoverageStatTests:
                    CoverageStatTests[expName] = {}
                if "random" in finalCoverage:
                    CoverageStatTests[expName]["mwu_" + expName + "_random"] = statTests.mwu_test_p(finalCoverage[expName], finalCoverage["random"])
                if "BonusMaxRL" in finalCoverage:
                    CoverageStatTests[expName]["mwu_" + expName + "_BonusMaxRL"] = statTests.mwu_test_p(finalCoverage[expName], finalCoverage["BonusMaxRL"])
                if "NegRLVisits" in finalCoverage:
                    CoverageStatTests[expName]["mwu_" + expName + "_NegRLVisits"] = statTests.mwu_test_p(finalCoverage[expName], finalCoverage["NegRLVisits"])

        # create the plot with Standard Deviation area
        p = plot.multilinePlotShortestCustomSD(avgDataSets, stdDevsSets, 1, "", 0, horizon, "upper left")
        p.savefig(targetFolder + "/" + expFolder.replace(" ","_") + "_SD" + ".png", bbox_inches="tight")
        p.savefig(targetFolder + "/" + expFolder.replace(" ","_") + "_SD" +".pdf", bbox_inches="tight")
        p.close()

        # create the plot
        p = plot.multilinePlotShortestCustom(avgDataSets, "", 0, horizon, "upper left")
        p.savefig(targetFolder + "/" + expFolder.replace(" ","_") + ".png", bbox_inches="tight")
        p.savefig(targetFolder + "/" + expFolder.replace(" ","_") + ".pdf", bbox_inches="tight")
        p.close()

        # save the final coverage data in a json file
        final = {}
        final["coverage"] = finalCoverage 
        final["coverageAvg"] = finalCoverageAvg 
        final["coverageSD"] = finalCoverageSD
        final["statTests"] = CoverageStatTests
        json_manager.save_json_file(targetFolder + "/" + expFolder.replace(" ","_") + "_final.json", final)


