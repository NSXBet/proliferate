import logging
import os
import git
from github import Github

logger = logging.getLogger(__name__)

def setup_repo(repo_path, config):
    """Configure git repository with author information"""
    repo_instance = git.Repo(repo_path)
    with repo_instance.config_writer() as git_config:
        git_config.set_value('user', 'name', config.get('author-name', 'SRE Team'))
        git_config.set_value('user', 'email', config.get('author-email', 'sre@nsx.bet'))
    return repo_instance

def create_or_update_branch(repo_instance, branch_name):
    """Create new branch or update existing one"""
    try:
        repo_instance.git.checkout(branch_name)
        logger.info(f"Updating existing branch: {branch_name}")
        try:
            repo_instance.git.pull('origin', branch_name)
        except git.GitCommandError:
            pass
    except git.GitCommandError:
        logger.info(f"Creating new branch: {branch_name}")
        repo_instance.git.checkout('-b', branch_name)

def create_or_update_pr(gh_repo, branch_name, pr_config):
    """Create new PR or update existing one"""
    existing_pr = None
    for pr in gh_repo.get_pulls(state='open'):
        if pr.head.ref == branch_name:
            existing_pr = pr
            break
    
    if existing_pr:
        existing_pr.edit(
            title=pr_config['pr-title'],
            body=pr_config['pr-body']
        )
        pr = existing_pr
        logger.info(f"Updated existing PR: {pr.html_url}")
    else:
        pr = gh_repo.create_pull(
            title=pr_config['pr-title'],
            body=pr_config['pr-body'],
            head=branch_name,
            base='main'
        )
        logger.info(f"Created new PR: {pr.html_url}")
    
    if 'pr-labels' in pr_config:
        pr.add_to_labels(*pr_config['pr-labels'])
    if 'pr-assignees' in pr_config:
        pr.add_to_assignees(*pr_config['pr-assignees'])
    
    return pr 