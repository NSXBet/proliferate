def patch(config, context):
    """
    Generate patch configuration for adding team metadata
    
    Args:
        config: Global configuration dictionary containing full repository path
        context: Dictionary containing repository-specific contexts
    
    Returns:
        dict: Patch configuration including branch, commit message, PR details, etc.
    """
    # Get repository name from full path (e.g., "sample-app" from "nsxbet/sample-app")
    repo_name = config['repo'].split('/')[-1]
    
    # Get repository-specific context
    repo_context = context.get(repo_name)
    if not repo_context:
        raise ValueError(f"No context found for repository: {repo_name}")
    
    # Get required values from context
    top_contributor = repo_context.get('top_contributor')
    if not top_contributor:
        raise ValueError(f"No top contributor found for repository: {repo_name}")
    
    # Get required values from context
    predominant_team = repo_context.get('predominant_team')
    if not predominant_team:
        raise ValueError(f"No predominant_team found for repository: {repo_name}")
    
    # Create PR body with contributor info
    pr_body = f"""This PR adds metadata to kubernetes configuration.

Repository's top contributor: {top_contributor}
"""
    
    return {
        'branch': 'sre/add-team-metadata',
        'commit-message': 'feat: add team metadata to kubernetes config',
        'pr-title': '[Automated] Add team metadata to kubernetes config',
        'pr-body': pr_body,
        'pr-labels': ['sre'],
        'pr-assignees': [top_contributor],
        'scripts_context': {
            'REPOSITORY': repo_name,
            "TEAM_NAME": predominant_team
        },
        'scripts': [
            'python /workspace/scripts/add-team-metadata.py'
        ]
    }
