class MysqlMcpServer < Formula
  desc "MySQL MCP Server - Model Context Protocol server for MySQL databases"
  homepage "https://github.com/askdba/mysql-mcp-server"
  version "1.7.0-rc.1"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/askdba/mysql-mcp-server/releases/download/v1.7.0-rc.1/mysql-mcp-server_1.7.0-rc.1_darwin_amd64.tar.gz"
      sha256 "2c7a09f9a56f557f5ada6391dc5b6c908baaa064ec54f085071d948b4659068d"
    end
    on_arm do
      url "https://github.com/askdba/mysql-mcp-server/releases/download/v1.7.0-rc.1/mysql-mcp-server_1.7.0-rc.1_darwin_arm64.tar.gz"
      sha256 "7c30db0d8673bfabcefd9ac99e1d829fd400028b98ed3c3508df379fcd2fe568"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/askdba/mysql-mcp-server/releases/download/v1.7.0-rc.1/mysql-mcp-server_1.7.0-rc.1_linux_amd64.tar.gz"
      sha256 "83362dd371a62467f7ce9d508ab58d8f97c211fa3e5804172403061bea481bcc"
    end
    on_arm do
      url "https://github.com/askdba/mysql-mcp-server/releases/download/v1.7.0-rc.1/mysql-mcp-server_1.7.0-rc.1_linux_arm64.tar.gz"
      sha256 "59a57c2917454f5151c22c3a0dcb7c09bd243e8496d3bad5f7082daed1021591"
    end
  end

  def install
    bin.install "mysql-mcp-server"
  end

  def caveats
    <<~EOS
      To use mysql-mcp-server with Claude Desktop, add to your config:

        {
          "mcpServers": {
            "mysql": {
              "command": "#{opt_bin}/mysql-mcp-server",
              "env": {
                "MYSQL_DSN": "user:password@tcp(localhost:3306)/database"
              }
            }
          }
        }

      Config location:
        macOS: ~/Library/Application Support/Claude/claude_desktop_config.json
        Linux: ~/.config/Claude/claude_desktop_config.json

      REST API: set MYSQL_MCP_HTTP=1 (and MYSQL_DSN). Token dashboard http://localhost:9306/status is on by default; MYSQL_MCP_TOKEN_CARD=0 disables it.
    EOS
  end

  test do
    # Basic test - server should fail without MYSQL_DSN but show proper error
    output = shell_output("#{bin}/mysql-mcp-server 2>&1", 1)
    assert_match(/MYSQL_DSN|config error/i, output)
  end
end
