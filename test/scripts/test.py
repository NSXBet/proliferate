#!/usr/bin/env python3

import os

VERSION_FILE = "VERSION"

def increment_version(version):
    return str(int(version) + 1)

def main():
    # Check if VERSION file exists
    if os.path.isfile(VERSION_FILE):
        # Read and increment version
        with open(VERSION_FILE, 'r') as f:
            current_version = f.read().strip()
        
        new_version = increment_version(current_version)
        
        with open(VERSION_FILE, 'w') as f:
            f.write(new_version)
        
        print(f"Incremented version from {current_version} to {new_version}")
    else:
        # Create VERSION file with initial version
        with open(VERSION_FILE, 'w') as f:
            f.write("1")
        print("Created VERSION file with initial version 1")

if __name__ == "__main__":
    main()