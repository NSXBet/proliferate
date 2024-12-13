import logging
from pathlib import Path
from ruamel.yaml import YAML
import importlib.util

logger = logging.getLogger(__name__)

def load_yaml(file_path):
    """Load YAML file using ruamel.yaml"""
    if not Path(file_path).exists():
        logger.error(f"File not found: {file_path}")
        return None
        
    yaml = YAML()
    yaml.preserve_quotes = True
    yaml.indent(mapping=2, sequence=4, offset=2)
    
    with open(file_path, 'r') as f:
        return yaml.load(f)

def load_patch_module(patch_path):
    """Dynamically load the patch module"""
    if not Path(patch_path).exists():
        logger.error(f"Patch file not found: {patch_path}")
        return None
        
    spec = importlib.util.spec_from_file_location("patch_module", patch_path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module 