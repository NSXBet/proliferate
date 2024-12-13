def patch(config, context):
    """
    Generate patch configuration for normalizing YAML indentation
    
    Args:
        config: Global configuration dictionary containing full repository path
        context: Dictionary containing repository-specific contexts
    
    Returns:
        dict: Patch configuration including branch, commit message, PR details, etc.
    """
    # Get repository name from full path
    repo_name = config['repo'].split('/')[-1]
    
    # Get repository-specific context
    repo_context = context.get(repo_name)
    if not repo_context:
        raise ValueError(f"No context found for repository: {repo_name}")
    
    # Get required values from context
    top_contributor = repo_context.get('top_contributor')
    if not top_contributor:
        raise ValueError(f"No top contributor found for repository: {repo_name}")
    
    pr_body = f"""This PR normalizes YAML indentation in configuration files.

Repository's top contributor: {top_contributor}
"""
    
    return {
        'branch': 'sre/normalize-indent',
        'commit-message': 'style: normalize yaml indentation',
        'pr-title': '[Automated] Normalize YAML indentation',
        'pr-body': pr_body,
        'pr-labels': ['sre'],
        'pr-assignees': [top_contributor],
        'scripts_context': {},
        'scripts': [
            '/workspace/scripts/normalize_indent.sh'
        ]
    } 