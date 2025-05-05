import unittest
import json
from file_provider import get_ssh_connection_info

class TestFileProvider(unittest.TestCase):

    def test_get_ssh_connection_info_with_cluster(self):
        target = "dario1@cluster1"
        expected_output = {
            "cluster": "cluster1",
            "host": "dario1",
            "port": 22,
            "username": "dario",
            "password": "password"
        }
        result = get_ssh_connection_info(target)
        self.assertEqual(json.loads(result), expected_output)

    def test_get_ssh_connection_info_without_cluster(self):
        target = "dario1"
        expected_output = {
            "cluster": "",
            "host": "dario1",
            "port": 22,
            "username": "dario",
            "password": "password"
        }
        result = get_ssh_connection_info(target)
        self.assertEqual(json.loads(result), expected_output)

    def test_get_ssh_connection_info_invalid_input(self):
        target = ""
        expected_output = {
            "cluster": "",
            "host": "",
            "port": 22,
            "username": "dario",
            "password": "password"
        }
        result = get_ssh_connection_info(target)
        self.assertEqual(json.loads(result), expected_output)

if __name__ == "__main__":
    unittest.main()