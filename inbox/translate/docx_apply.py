#!/usr/bin/env python3
"""Reinsere traducoes no .docx alterando somente o conteudo dos <w:t>.

Todas as outras entradas do pacote OPC (imagens, estilos, numeracao, temas,
relacionamentos, [Content_Types].xml) sao copiadas byte-a-byte, preservando
tambem o metodo de compressao original de cada entrada.
"""
import json
import shutil
import sys
import zipfile

from docx_seg import segment_part, text_parts

ESC = ((b"&", b"&amp;"), (b"<", b"&lt;"), (b">", b"&gt;"))


def esc(s: str) -> bytes:
    b = s.encode("utf-8")
    for a, r in ESC:
        b = b.replace(a, r)
    return b


def apply_part(xml: bytes, trans: dict, stats: dict) -> bytes:
    segs = segment_part(xml)
    edits = []
    for s in segs:
        new = trans.get(s["text"])
        if new is None or new == s["text"]:
            continue
        runs = s["runs"]
        first = runs[0]
        c_start, c_end, t_start, t_end = first
        # texto traduzido inteiro vai para o primeiro <w:t> do segmento
        edits.append((c_start, c_end, esc(new)))
        # os demais <w:t> do segmento ficam vazios (run preservado, sem texto)
        for c_s, c_e, _, _ in runs[1:]:
            edits.append((c_s, c_e, b""))
        # garantir xml:space="preserve" quando ha espaco nas bordas
        if new[:1].isspace() or new[-1:].isspace():
            tag = xml[t_start:t_end]
            if b"xml:space" not in tag:
                edits.append((t_end - 1, t_end - 1, b' xml:space="preserve"'))
        stats["segmentos_alterados"] += 1

    # aplicar de tras para frente para nao invalidar offsets
    edits.sort(key=lambda e: (e[0], e[1]), reverse=True)
    out = bytearray(xml)
    for start, end, repl in edits:
        out[start:end] = repl
    return bytes(out)


def main(src, dst, trans_path):
    trans = {}
    order = json.load(open("segments.json"))["order"]
    raw = json.load(open(trans_path))
    for i, s in enumerate(order):
        v = raw.get(str(i))
        if v is not None:
            trans[s] = v

    zin = zipfile.ZipFile(src)
    parts = set(text_parts(zin))
    stats = {"segmentos_alterados": 0, "partes_alteradas": 0}

    zout = zipfile.ZipFile(dst, "w", zipfile.ZIP_DEFLATED, allowZip64=True)
    for info in zin.infolist():
        data = zin.read(info.filename)
        if info.filename in parts:
            before = data
            data = apply_part(data, trans, stats)
            if data != before:
                stats["partes_alteradas"] += 1
        # preservar metadados da entrada original
        ni = zipfile.ZipInfo(info.filename, date_time=info.date_time)
        ni.compress_type = info.compress_type
        ni.external_attr = info.external_attr
        ni.internal_attr = info.internal_attr
        ni.create_system = info.create_system
        zout.writestr(ni, data)
    zout.close()
    zin.close()
    print(f"partes alteradas: {stats['partes_alteradas']}")
    print(f"segmentos alterados: {stats['segmentos_alterados']}")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2], sys.argv[3])
