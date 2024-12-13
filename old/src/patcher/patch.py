import logging
import os
import tempfile
import subprocess
import git
from . import git as git_utils
from github import Github

logger = logging.getLogger(__name__)

async def apply_patch_to_repo(repo, patch_module, config, context, dry_run=False):
    """Apply patch to a single repository"""
    logger.info(f"Applying patch to repository: {repo} {'(dry run)' if dry_run else ''}")

    temp_dir = tempfile.mkdtemp()
    try:
        # Clone and setup repository
        github_token = os.environ.get('GITHUB_TOKEN')
        if not github_token:
            raise ValueError("GITHUB_TOKEN environment variable is required")
            
        repo_url = f"https://{github_token}@github.com/{repo}.git"
        repo_path = os.path.join(temp_dir, repo.split('/')[-1])
        repo_instance = git.Repo.clone_from(repo_url, repo_path)
        git_utils.setup_repo(repo_path, config)
        
        # Apply patch and execute scripts
        result = await execute_patch(repo_instance, repo, patch_module, config, context, dry_run)
        result['temp_dir'] = temp_dir
        return result
        
    except Exception as e:
        return {
            'repo': repo,
            'status': 'error',
            'error': str(e),
            'temp_dir': temp_dir
        }

async def execute_patch(repo_instance, repo, patch_module, config, context, dry_run):
    """Execute patch and handle results"""
    result = patch_module.patch({ 'repo': repo, 'dry_run': dry_run }, context)
    
    original_dir = os.getcwd()
    os.chdir(repo_instance.working_dir)
    
    try:
        # Execute scripts
        script_env = os.environ.copy()
        script_env.update(result.get('scripts_context', {}))
        
        for script in result.get('scripts', []):
            subprocess.run(script, shell=True, env=script_env, check=True)
        
        # Check for changes
        diff = repo_instance.git.diff()
        if not diff:
            return {
                'repo': repo,
                'status': 'error',
                'error': 'No changes detected after applying patch'
            }
        
        logger.info(f"Changes detected:\n{diff}")
        
        if dry_run:
            logger.info("Dry run - skipping commit and push")
            return {
                'repo': repo,
                'status': 'would_change',
                'result': result,
                'diff': diff
            }
        
        # Handle git operations
        return await handle_git_operations(repo_instance, repo, result, diff)
        
    finally:
        os.chdir(original_dir)

async def handle_git_operations(repo_instance, repo, result, diff):
    """Handle git operations and PR creation"""
    branch_name = result['branch']
    git_utils.create_or_update_branch(repo_instance, branch_name)
    
    repo_instance.git.add('.')
    repo_instance.git.commit('-m', result['commit-message'])
    repo_instance.git.push('origin', branch_name, '--force')
    
    gh = Github(os.environ['GITHUB_TOKEN'])
    gh_repo = gh.get_repo(repo)
    pr = git_utils.create_or_update_pr(gh_repo, branch_name, result)
    
    return {
        'repo': repo,
        'status': 'success',
        'result': result,
        'diff': diff,
        'pr_number': pr.number,
        'pr_url': pr.html_url,
        'pr_action': 'updated' if pr.head.ref == branch_name else 'created'
    } 