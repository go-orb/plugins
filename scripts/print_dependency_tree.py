#!/usr/bin/env python3

import os
import re
from collections import defaultdict
from pathlib import Path

def get_module_name(path):
    with open(path, "r") as file:
        regex = re.compile(r"module\s+(.*)")
        for line in file:
            match = regex.match(line)
            if match:
                return match[1]
    return None

def find_module_file(module_name, search_dir="."):
    """Find the go.mod file for a given module name."""
    for root, _, files in os.walk(search_dir):
        if 'go.mod' in files:
            mod_file = os.path.join(root, 'go.mod')
            if get_module_name(mod_file) == module_name:
                return mod_file
    return None

def parse_dependencies(mod_file):
    """Parse dependencies from a go.mod file."""
    regex = re.compile(r"^\s+(github\.com\/go-orb\/plugins\/[\w\/]+)\s")
    with open(mod_file, 'r') as file:
        lines = file.readlines()
    dependencies = [regex.match(line)[1] for line in lines if regex.match(line)]
    return dependencies

def build_dependency_graph(search_dir="."):
    """Build a complete dependency graph by recursively parsing go.mod files."""
    dependency_graph = defaultdict(list)
    processed_modules = set()

    def process_module(module_name):
        if module_name in processed_modules:
            return
        
        processed_modules.add(module_name)
        mod_file = find_module_file(module_name, search_dir)
        
        if mod_file:
            dependencies = parse_dependencies(mod_file)
            dependency_graph[module_name].extend(dependencies)
            
            # Recursively process each dependency
            for dep in dependencies:
                process_module(dep)

    # Find all go.mod files and process them
    for root, _, files in os.walk(search_dir):
        if root.endswith('/.github'):
            continue

        if 'go.mod' in files:
            mod_file = os.path.join(root, 'go.mod')
            module_name = get_module_name(mod_file)
            if module_name:
                process_module(module_name)

    return dependency_graph

def print_dependency_tree(graph, node, level=0, visited=None):
    """Recursively print the dependency tree."""
    if visited is None:
        visited = set()
    
    if node in visited:
        print(f"  {'- ' * level}{node} (circular dependency)")
        return
    
    visited.add(node)
    print(f"  {'- ' * level}{node}")

    for dep in graph.get(node, []):
        print_dependency_tree(graph, dep, level + 1, visited.copy())


def main():
    # Build the complete dependency graph
    dependency_graph = build_dependency_graph(str(Path(__file__).parent.parent))

    # Print the dependency graph as a tree
    all_deps = {dep for deps in dependency_graph.values() for dep in deps}
    roots = set(dependency_graph.keys()) - all_deps
    
    print("\nDependency Tree:")
    for root in sorted(roots):
        print_dependency_tree(dependency_graph, root)

if __name__ == "__main__":
    main()