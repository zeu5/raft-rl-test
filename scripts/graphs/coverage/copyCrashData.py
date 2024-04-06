import sys
from os import listdir
import os
import shutil
from distutils.dir_util import copy_tree

targetFolder = sys.argv[1]
destFolder = sys.argv[2]

experimentsFolder = targetFolder + "/experiments"

for file in listdir(experimentsFolder): # foreach experiment folder
    if os.path.isdir(experimentsFolder + "/" + file):
        # build the experiment tag
        subParts = file.split("_")
        machineName = subParts[2]
        experimentName = subParts[3]
        experimentName = machineName + "_" + experimentName[0] + "_" + experimentName[-1]

    expFolder = experimentsFolder + "/" + file
    copy_tree(expFolder + "/bugs", destFolder + "/" + experimentName + "/bugs")
    copy_tree(expFolder + "/crash", destFolder + "/" + experimentName + "/crash")
    