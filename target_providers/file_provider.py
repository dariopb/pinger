from mcp.server.fastmcp import FastMCP
import json
import jsonpickle
import sys

from config import parse_arguments, load_config

mcp = FastMCP("file-provider")

class Cluster:
    def __init__(self, name: str, nodes: list[str]):
        self.name = name
        self.nodes = nodes


@mcp.tool()
def get_all_nodes() -> str:
    """Get all endpoints."""

    node_names = []
    for endpoint in config.nodes:
        node_names.append(endpoint.name)

    clusters = [ Cluster( "Local", node_names) ]
    text = jsonpickle.encode(clusters, unpicklable=False)

    return text

@mcp.tool(name="get_ssh_connection_info",
          description="Get the endpoint data for a target host to connect via SSH. Input is like this: dario1@cluster1")
def get_ssh_connection_info(target: str) -> str:
    """Get the endpoint data for a target host to connect via SSH. Input is like this: dario1@cluster1"""
    print("get_ssh_connection_info: ", target)

    cluster = target.split("@")
    if len(cluster) == 2:
        host_name = cluster[0]
        cluster_name = cluster[1]
    else:
        host_name = target
        cluster_name = ""

    # create and return an object as a json string with the endpoint data
    node = None
    for endpoint in config.nodes:
        # Check if this is the right node
        if endpoint.name == host_name:
            node = endpoint
            break
            
    if node is None:
        print(f"Warning: Node {host_name} not found in configuration")
        return ""

    endpoint_data = {
        "cluster": "Local",
        "host": node.name,
        "port": 22,
        "target": node.target,
        "username": node.username,
        "password": "",
        "key": node.key,
    }
    # convert the object to a json string
    endpoint_data_json = json.dumps(endpoint_data)
    print("endpoint_data_json: ", endpoint_data_json)

    return endpoint_data_json


args = parse_arguments()

# Load the configuration into simple AppConfig object
config = load_config(args.config)
print(f"Configuration loaded successfully from {args.config}")



def main():
    print("Starting file-provider!")

    clusters = get_all_nodes()
    print("Clusters: ", clusters)

    ssh_endpoint = get_ssh_connection_info("localhost@local")
    print("SSH Endpoint: ", ssh_endpoint)

    # get_ssh_connection_info("dario1@cluster1")


if __name__ == "__main__":
    print("Starting file-provider")
    print("Command-line arguments:", sys.argv)

    mcp.run()
    main()

