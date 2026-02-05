#!/usr/bin/env bash
# Welcome screen - First-time user experience
# ============================================================================

# ============================================================================
# WELCOME MESSAGES
# ============================================================================

# Show first-time welcome message
show_first_time_welcome() {
    logo
    printf '\n'
    cecho "Welcome to Agentbox!" "$GREEN"
    printf '\n'
    printf '%s\n' "Agentbox runs AI coding assistants in isolated Docker containers."
    printf '\n'
    printf '%s\n' "Supported agents:"
    printf '  • Claude Code (claude)   - Anthropic'\''s AI assistant\n'
    printf '  • Codex (codex)          - OpenAI'\''s coding assistant\n'
    printf '  • OpenCode (opencode)    - Open-source coding assistant\n'
    printf '\n'
    printf '%s\n' "Get started:"
    printf '\n'
    printf '  1. Enable an agent:\n'
    printf "     ${CYAN}agentbox enable claude${NC}\n"
    printf '\n'
    printf '  2. Navigate to your project:\n'
    printf "     ${CYAN}cd /path/to/your/project${NC}\n"
    printf '\n'
    printf '  3. Run the agent:\n'
    printf "     ${CYAN}agentbox claude${NC}\n"
    printf '\n'
    printf '%s\n' "Useful commands:"
    printf "  ${CYAN}%-25s${NC} - %s\n" "agentbox list" "Show available agents"
    printf "  ${CYAN}%-25s${NC} - %s\n" "agentbox help" "Show all commands"
    printf "  ${CYAN}%-25s${NC} - %s\n" "agentbox aliases" "Get shell aliases"
    printf '\n'
}

# Show no-agent-enabled message
show_no_agent_message() {
    logo_small
    printf '\n'
    cecho "No agents enabled" "$YELLOW"
    printf '\n'
    printf '%s\n' "Enable an agent to get started:"
    printf '\n'
    printf "  ${CYAN}agentbox enable claude${NC}     # Enable Claude Code\n"
    printf "  ${CYAN}agentbox enable codex${NC}      # Enable OpenAI Codex\n"
    printf "  ${CYAN}agentbox enable opencode${NC}   # Enable OpenCode\n"
    printf '\n'
    printf '%s\n' "Or list available agents:"
    printf "  ${CYAN}agentbox list${NC}\n"
    printf '\n'
}

# Show project not initialized message
show_project_init_message() {
    logo_small
    printf '\n'
    cecho "Project not initialized" "$YELLOW"
    printf '\n'
    printf '%s\n' "Run an agent in this directory to initialize it:"
    printf '\n'
    printf "  ${CYAN}agentbox claude${NC}\n"
    printf '\n'
    printf '%s\n' "This will:"
    printf '  • Create a containerized environment\n'
    printf '  • Mount your project directory\n'
    printf '  • Set up agent configuration\n'
    printf '\n'
}

# ============================================================================
# EXPORTS
# ============================================================================

export -f show_first_time_welcome show_no_agent_message show_project_init_message
