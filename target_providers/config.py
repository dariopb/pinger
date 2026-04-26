import argparse
import json
import os
import sys
from dataclasses import dataclass, field
from typing import Dict

#from target_providers.file_provider import SshEndpoint

   
class SshEndpoint:
    def __init__(self, name, target, username, password, key):
        self.name = name
        self.target = target
        self.username = username
        self.password = password
        self.key = key

    def __repr__(self):
        return f"SshEndpoint(name={self.name}, target={self.target}, username={self.username}, password={self.password}, key={self.key})"


def parse_arguments():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(description='Load configuration and certificate files')
    
    parser.add_argument('--config', '-c', 
                        required=True,
                        help='Path to the JSON configuration file')
    
    return parser.parse_args()


@dataclass
class AppConfig:
    """Simple configuration class with a hashmap for settings."""
    nodes: list[SshEndpoint]

    def __init__(self, nodes: list[SshEndpoint]):
        self.nodes = nodes


def load_config(config_path) -> AppConfig:
    """Load and parse the JSON configuration file into AppConfig object."""
    try:
        if not os.path.exists(config_path):
            raise FileNotFoundError(f"Configuration file not found: {config_path}")
        
        with open(config_path, 'r') as file:
            json_data = load_file_content(file_path=config_path)
            nodes = parse_config_file(json_data)
            
            return AppConfig(nodes=nodes)
    
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON configuration: {e}", file=sys.stderr)
        sys.exit(1)
    
    except Exception as e:
        print(f"Error loading configuration: {e}", file=sys.stderr)
        sys.exit(1)


def parse_config_file(hosts_json: str) -> str:
    """Parse a JSON hosts file containing SSH connection information"""
    try:
        data = json.loads(hosts_json)
        ssh_endpoints = []
        
        if "nodes" in data:
            for node in data["nodes"]:
                key = load_file_content(node.get("key", ""))
                ssh_endpoint = SshEndpoint(
                    name=node.get("name", ""),
                    target=node.get("target", ""),
                    username=node.get("username", ""),
                    password=node.get("password", ""),
                    key=key
                )
                ssh_endpoints.append(ssh_endpoint)
        
        return ssh_endpoints
    except json.JSONDecodeError as e:
        return f"Error parsing JSON: {str(e)}"
    except Exception as e:
        return f"Error: {str(e)}"



def load_file_content(file_path):
    """Load the content of a file."""
    try:
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"File not found: {file_path}")
        
        with open(file_path, 'r') as file:
            content = file.read()
        
        return content
    
    except Exception as e:
        print(f"Error loading file {file_path}: {e}", file=sys.stderr)
        sys.exit(1)
