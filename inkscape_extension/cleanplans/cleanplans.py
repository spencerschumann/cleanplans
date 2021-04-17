#!/usr/bin/env python3

import sys
import os
import subprocess
import platform

if __name__ == '__main__':
    infile = sys.argv[1]
    system = platform.system()
    path = os.path.realpath(os.path.dirname(__file__))
    command = ''
    if system == 'Linux':
        command = 'cleanplans_linux'
    elif system == 'Windows':
        command = 'cleanplans.exe'
    elif system == 'Darwin':
        command = 'cleanplans_osx'
    result = subprocess.run([os.path.join(path, command), infile], capture_output=True)
    print(result.stdout.decode())
