#!/usr/bin/env python3

import sys
import os
from pathlib import Path
from ruamel.yaml import YAML

def find_yaml_files(start_path):
    """Recursively search for staging.yaml and production.yaml"""
    yaml_files = []
    for root, _, files in os.walk(start_path):
        for file in ['staging.yaml', 'production.yaml']:
            if file in files:
                yaml_files.append(Path(root) / file)
    return yaml_files

def add_team_metadata():
    # Get required environment variables
    repository = os.getenv('REPOSITORY')
    team_name = os.getenv('TEAM_NAME')
    
    if not repository:
        print("Error: REPOSITORY environment variable not set")
        sys.exit(1)
    
    if not team_name:
        print("Error: TEAM_NAME environment variable not set")
        sys.exit(1)
    
    # Print debug info
    print(f"Processing repository: {repository}")
    print(f"Team name: {team_name}")
    
    # Initialize ruamel.yaml with minimal formatting
    yaml = YAML()
    yaml.preserve_quotes = True
    yaml.width = 4096  # Prevent line wrapping
    yaml.indent(mapping=None, sequence=None, offset=None)
    yaml.explicit_start = False
    yaml.explicit_end = False
    
    # Search for yaml files recursively
    yaml_files = find_yaml_files(os.getcwd())
    if not yaml_files:
        print("No staging.yaml or production.yaml found in any subdirectory")
        return
    
    for yaml_file in yaml_files:
        print(f"\nProcessing: {yaml_file}")
        
        # Read the yaml file
        try:
            with open(yaml_file, 'r') as file:
                data = yaml.load(file)
        except Exception as e:
            print(f"Error reading YAML file: {e}")
            continue
        
        # Check if kubernetes key exists
        if data and isinstance(data, dict) and 'kubernetes' in data:
            # Initialize .kubernetes.base.metadata if it doesn't exist
            if 'base' not in data['kubernetes']:
                data['kubernetes']['base'] = {}
            if 'metadata' not in data['kubernetes']['base']:
                data['kubernetes']['base']['metadata'] = {}
            
            # Add team metadata
            data['kubernetes']['base']['metadata']['team'] = team_name
            
            # Write back to the file
            try:
                with open(yaml_file, 'w') as file:
                    yaml.dump(data, file)
                print(f"Successfully updated {yaml_file}")
            except Exception as e:
                print(f"Error writing YAML file: {e}")
                continue
        else:
            print(f"No kubernetes configuration found in {yaml_file}")

if __name__ == "__main__":
    add_team_metadata() 