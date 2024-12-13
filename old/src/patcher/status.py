import logging
import json
import asyncio
from pathlib import Path
from github import Github
from github.GithubException import GithubException
import os

logger = logging.getLogger(__name__)

async def check_single_pr(gh, info):
    """Check status of a single PR"""
    try:
        repo_obj = gh.get_repo(info['repo'])
        pr = repo_obj.get_pull(info['pr_number'])
        
        status = "MERGED" if pr.merged else pr.state.upper()
        reviews = len(list(pr.get_reviews()))
        
        return {
            'success': True,
            'info': info,
            'status': status,
            'reviews': reviews,
            'updated_at': pr.updated_at,
            'mergeable': pr.mergeable if not pr.merged else None
        }
        
    except Exception as e:
        return {
            'success': False,
            'info': info,
            'error': str(e)
        }

def print_pr_status(key, result):
    """Print formatted PR status"""
    print("-" * 80)
    info = result['info']
    
    if result['success']:
        print(f"\nPatch: {info['patch']}")
        print(f"PR Key: {key}")
        print(f"Repository: {info['repo']}")
        print(f"PR: {info['pr_url']}")
        print(f"Status: {result['status']}")
        print(f"Reviews: {result['reviews']}")
        print(f"Last Updated: {result['updated_at']}")
        if result['mergeable'] is not None:
            print(f"Mergeable: {result['mergeable']}")
    else:
        print(f"\nPR Key: {key}")
        print(f"Repository: {info['repo']}")
        print(f"Patch: {info['patch']}")
        print(f"Error: {result['error']}")

async def check_pull_requests():
    """Check status of pull requests from history"""
    history_file = Path('.patcher/pr_history.json')
    if not history_file.exists():
        logger.error("No PR history found")
        return
    
    # Initialize GitHub client
    github_token = os.environ.get('GITHUB_TOKEN')
    if not github_token:
        logger.error("GITHUB_TOKEN environment variable is required")
        return
    
    gh = Github(github_token)
    
    # Load PR history
    with open(history_file) as f:
        history = json.load(f)
    
    print("\nPull Request Status Report:")
    
    # Create tasks for all PRs
    tasks = [check_single_pr(gh, info) for key, info in history.items()]
    results = await asyncio.gather(*tasks)
    
    # Print results in order
    for (key, _), result in zip(history.items(), results):
        print_pr_status(key, result)