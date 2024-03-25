import sys
from os import listdir
import os
import shutil

def copyToFolder(sourceFile, destFile, targetFolder):
    if not os.path.exists(targetFolder):
        os.makedirs(targetFolder)
    shutil.copyfile(sourceFile, targetFolder + "/" + destFile)

targetFolder = sys.argv[1]

experimentsFolder = targetFolder + "/experiments"
destinationsFolder = targetFolder + "/coverageData"
if not os.path.exists(destinationsFolder):
    os.makedirs(destinationsFolder)

for file in listdir(experimentsFolder): # foreach experiment folder
    if os.path.isdir(experimentsFolder + "/" + file):
        # build the experiment tag
        subParts = file.split("_")
        machineName = subParts[2]
        experimentName = subParts[3]
        experimentName = machineName + "_" + experimentName[0] + "_" + experimentName[-1]

    coverageFolder = experimentsFolder + "/" + file + "/coverage"
    for coverageFile in listdir(coverageFolder):
        if coverageFile == "0_data.json": # generic coverage
            copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destinationsFolder + "/generic")
        elif coverageFile.endswith("[1]0_data.json"):
            specificCoverageName = coverageFile.replace("[1]0_data.json", "")
            copyToFolder(coverageFolder + "/" + coverageFile, experimentName + ".json", destinationsFolder + "/" + specificCoverageName)
        
    