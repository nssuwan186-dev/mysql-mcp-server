#!/bin/bash
# MySQL MCP Server - Quickstart Script
# This script helps you get started with mysql-mcp-server

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║           MySQL MCP Server - Quickstart                   ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check for required tools
check_requirements() {
    echo -e "${YELLOW}Checking requirements...${NC}"
    
    if ! command -v mysql &> /dev/null; then
        echo -e "${RED}MySQL client not found. Please install MySQL.${NC}"
        echo "  macOS: brew install mysql"
        echo "  Ubuntu: sudo apt install mysql-client"
        exit 1
    fi
    echo -e "${GREEN}✓ MySQL client found${NC}"
}

# Get MySQL connection details
get_mysql_config() {
    echo ""
    echo -e "${YELLOW}MySQL Connection Setup${NC}"
    echo "Enter your MySQL connection details:"
    echo ""
    
    read -p "Host [localhost]: " MYSQL_HOST
    MYSQL_HOST=${MYSQL_HOST:-localhost}
    
    read -p "Port [3306]: " MYSQL_PORT
    MYSQL_PORT=${MYSQL_PORT:-3306}
    
    read -p "Username [root]: " MYSQL_USER
    MYSQL_USER=${MYSQL_USER:-root}
    
    read -sp "Password: " MYSQL_PASS
    echo ""
    
    read -p "Database [mysql]: " MYSQL_DB
    MYSQL_DB=${MYSQL_DB:-mysql}
    
    # Build DSN
    MYSQL_DSN="${MYSQL_USER}:${MYSQL_PASS}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}?parseTime=true"
}

# Test MySQL connection
test_connection() {
    echo ""
    echo -e "${YELLOW}Testing MySQL connection...${NC}"
    
    if mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" -e "SELECT 1" &> /dev/null; then
        echo -e "${GREEN}✓ MySQL connection successful${NC}"
    else
        echo -e "${RED}✗ Failed to connect to MySQL${NC}"
        echo "Please check your credentials and try again."
        exit 1
    fi
}

# Create MCP user (optional)
create_mcp_user() {
    echo ""
    read -p "Create a dedicated MCP user with read-only access? [y/N]: " CREATE_USER
    
    if [[ "$CREATE_USER" =~ ^[Yy]$ ]]; then
        read -p "MCP Username [mcp]: " MCP_USER
        MCP_USER=${MCP_USER:-mcp}
        
        read -sp "MCP Password: " MCP_PASS
        echo ""
        
        echo -e "${YELLOW}Creating MCP user...${NC}"
        mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" <<EOF
CREATE USER IF NOT EXISTS '${MCP_USER}'@'%' IDENTIFIED BY '${MCP_PASS}';
GRANT SELECT ON *.* TO '${MCP_USER}'@'%';
FLUSH PRIVILEGES;
EOF
        echo -e "${GREEN}✓ MCP user created with SELECT-only privileges${NC}"
        
        # Update DSN to use MCP user
        MYSQL_DSN="${MCP_USER}:${MCP_PASS}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}?parseTime=true"
    fi
}

# Generate Claude Desktop config
generate_claude_config() {
    echo ""
    echo -e "${YELLOW}Generating Claude Desktop configuration...${NC}"
    
    BINARY_PATH=$(which mysql-mcp-server 2>/dev/null || echo "/path/to/mysql-mcp-server")
    
    CONFIG_FILE="claude_desktop_config.json"
    cat > "$CONFIG_FILE" <<EOF
{
  "mcpServers": {
    "mysql": {
      "command": "${BINARY_PATH}",
      "env": {
        "MYSQL_DSN": "${MYSQL_DSN}",
        "MYSQL_MAX_ROWS": "200",
        "MYSQL_QUERY_TIMEOUT_SECONDS": "30",
        "MYSQL_MCP_EXTENDED": "1"
      }
    }
  }
}
EOF
    
    echo -e "${GREEN}✓ Configuration saved to ${CONFIG_FILE}${NC}"
    echo ""
    echo "Copy this configuration to:"
    echo "  macOS: ~/Library/Application Support/Claude/claude_desktop_config.json"
    echo "  Linux: ~/.config/Claude/claude_desktop_config.json"
}

# Load test dataset
load_test_dataset() {
    echo ""
    read -p "Load example test dataset? [y/N]: " LOAD_DATA
    
    if [[ "$LOAD_DATA" =~ ^[Yy]$ ]]; then
        if [ -f "examples/test-dataset.sql" ]; then
            echo -e "${YELLOW}Loading test dataset...${NC}"
            mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" < examples/test-dataset.sql
            echo -e "${GREEN}✓ Test dataset loaded into 'mcp_demo' database${NC}"
        else
            echo -e "${RED}Test dataset file not found: examples/test-dataset.sql${NC}"
        fi
    fi
}

# Main
main() {
    check_requirements
    get_mysql_config
    test_connection
    create_mcp_user
    generate_claude_config
    load_test_dataset
    
    echo ""
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    Setup Complete! 🎉                     ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Copy ${CONFIG_FILE} to your Claude Desktop config location"
    echo "  2. Restart Claude Desktop"
    echo "  3. Ask Claude to explore your MySQL databases!"
    echo ""
    echo "Example prompts:"
    echo '  - "List all databases and their sizes"'
    echo '  - "Describe the schema of the users table"'
    echo '  - "Show me the top 10 largest tables"'
    echo ""
}

main "$@"

