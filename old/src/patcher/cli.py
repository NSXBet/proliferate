import argparse
import asyncio
import logging
import json
from pathlib import Path
from . import config
from .patch import apply_patch_to_repo
from .status import check_pull_requests

logger = logging.getLogger(__name__)

def save_run_results(results, patch_path):
    """Save run results to history file"""
    history_file = Path('.patcher/pr_history.json')
    history = {}
    
    # Load existing history if it exists
    if history_file.exists():
        try:
            with open(history_file) as f:
                history = json.load(f)
        except json.JSONDecodeError:
            logger.warning("Could not read PR history, starting fresh")
    
    # Get patch name from path
    patch_name = Path(patch_path).stem
    
    # Update history with new results
    for result in results:
        if result.get('status') == 'success' and 'pr_number' in result:
            # Create unique key using repo and PR number
            key = f"{result['repo']}-{result['pr_number']}"
            history[key] = {
                'repo': result['repo'],
                'pr_number': result['pr_number'],
                'pr_url': result['pr_url'],
                'patch': patch_name
            }
    
    # Save updated history
    Path('.patcher').mkdir(exist_ok=True)
    with open(history_file, 'w') as f:
        json.dump(history, f, indent=2)

async def batch_apply_patches(patch_path, context_path, dry_run=False):
    """Main function to apply patch to all repositories"""
    cfg = config.load_yaml('config.yaml')
    context = config.load_yaml(context_path)
    patch_module = config.load_patch_module(patch_path)
    
    if not all([cfg, context, patch_module]):
        logger.error("Missing required configuration files")
        return None
    
    logger.info(f"Loaded configuration files successfully")
    logger.info(f"Found {len(cfg['repo'])} repositories to process")
    if dry_run:
        logger.info("Running in dry-run mode - no changes will be pushed")
    
    tasks = [apply_patch_to_repo(repo, patch_module, cfg, context, dry_run) 
             for repo in cfg['repo']]
    results = await asyncio.gather(*tasks)
    
    # Save results to history
    if not dry_run:
        save_run_results(results, patch_path)
    
    logger.info("\nPatch Application Results:")
    print(json.dumps(results, indent=2))
    return results

def main():
    parser = argparse.ArgumentParser(description='Apply patch to repositories')
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # Apply patch command
    apply_parser = subparsers.add_parser('apply_patch', help='Apply a patch to repositories')
    apply_parser.add_argument('patch_path', help='Path to the patch file')
    apply_parser.add_argument('-c', '--context', required=True, help='Path to context YAML file')
    apply_parser.add_argument('--dry-run', action='store_true', help='Run without making changes')
    
    # Status command
    status_parser = subparsers.add_parser('status', help='Check status of pull requests')
    
    args = parser.parse_args()
    
    if args.command == 'apply_patch':
        asyncio.run(batch_apply_patches(args.patch_path, args.context, args.dry_run))
    elif args.command == 'status':
        asyncio.run(check_pull_requests()) 