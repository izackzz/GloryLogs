#!/usr/bin/env python3
import os
import math

# Threshold de 100 MB
CHUNK_SIZE = 100 * 1024 * 1024

# Pasta de logs relativa ao script
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
LOGS_DIR = os.path.join(SCRIPT_DIR, "logs")


def split_file(path: str):
    """
    Divide um arquivo em partes de até CHUNK_SIZE.
    """
    size = os.path.getsize(path)
    total_parts = math.ceil(size / CHUNK_SIZE)
    base, ext = os.path.splitext(path)

    with open(path, 'rb') as infile:
        for idx in range(1, total_parts + 1):
            chunk = infile.read(CHUNK_SIZE)
            if not chunk:
                break
            part_name = f"{base}-[{idx}-{total_parts}]{ext}"
            with open(part_name, 'wb') as outfile:
                outfile.write(chunk)
    os.remove(path)  # remove arquivo original após divisão
    print(f"Dividido: {os.path.basename(path)} em {total_parts} partes.")


def main():
    if not os.path.isdir(LOGS_DIR):
        print(f"Pasta não encontrada: {LOGS_DIR}")
        return

    for filename in os.listdir(LOGS_DIR):
        if not filename.lower().endswith('.txt'):
            continue

        full_path = os.path.join(LOGS_DIR, filename)
        if os.path.getsize(full_path) > CHUNK_SIZE:
            split_file(full_path)
    print("Processamento concluído.")


if __name__ == '__main__':
    main()
