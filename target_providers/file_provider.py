from mcp.server.fastmcp import FastMCP
import json


mcp = FastMCP("file-provider")

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
    endpoint_data = {
        "cluster": cluster_name,
        "host": host_name,
        "port": 22,
        "username": "dario",
        "password": "pablo207!!"
    }
    # convert the object to a json string
    endpoint_data_json = json.dumps(endpoint_data)
    print("endpoint_data_json: ", endpoint_data_json)

    return endpoint_data_json


def main():
    print("Hello from file-provider!")


if __name__ == "__main__":
    print("Starting file-provider")
    #get_endpoint_for("dario1cluster1")
    mcp.run()

