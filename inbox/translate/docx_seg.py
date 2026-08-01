#!/usr/bin/env python3
"""
Segmentacao e reinsercao de texto em .docx preservando layout byte-a-byte.

Estrategia: nunca reserializar o XML com um parser (isso renomeia prefixos de
namespace e invalida atributos como mc:Ignorable). Em vez disso, tokenizamos o
XML bruto, localizamos os intervalos de bytes do conteudo de cada <w:t>, e
reescrevemos apenas esses intervalos.

Um "segmento" e uma sequencia maximal de elementos <w:t> consecutivos, dentro
de um mesmo paragrafo, que compartilham o mesmo rPr (formatacao de caractere) e
nao possuem elementos de quebra entre si. Traduzir por segmento (em vez de por
<w:t>) da contexto de frase ao tradutor sem cruzar fronteiras de formatacao.
"""
import re
import sys
import json
import html
import zipfile

# Tags que interrompem um segmento: quebras, tabulacoes, desenhos, campos.
FLUSH_TAGS = (
    "br", "tab", "drawing", "object", "pict", "fldChar", "instrText",
    "delText", "footnoteReference", "endnoteReference", "commentReference",
    "ptab", "cr", "noBreakHyphen", "softHyphen", "sym",
)

TAG_RE = re.compile(rb"<(/?)([A-Za-z0-9_.\-]+:)?([A-Za-z0-9_.\-]+)([^>]*?)(/?)>", re.S)


def segment_part(xml: bytes):
    """Retorna lista de segmentos. Cada segmento e um dict com:
       'runs': lista de (start, end, tag_start, tag_end, has_preserve)
               onde start:end delimita o CONTEUDO textual do <w:t>
       'text': texto concatenado do segmento
    """
    segments = []
    cur_runs = []          # acumulador do segmento atual
    cur_key = None         # rPr do segmento atual
    run_rpr = None         # rPr do <w:r> corrente
    in_rpr = False
    rpr_start = None
    pending_t = None       # (content_start, tag_start, tag_end, has_preserve)

    def flush():
        nonlocal cur_runs, cur_key
        if cur_runs:
            text = "".join(r[4] for r in cur_runs)
            segments.append({"runs": [r[:4] for r in cur_runs], "text": text})
        cur_runs = []
        cur_key = None

    for m in TAG_RE.finditer(xml):
        closing = m.group(1) == b"/"
        local = m.group(3).decode("ascii", "replace")
        selfclose = m.group(5) == b"/"

        # --- rPr: capturar bloco bruto como chave de formatacao ---
        if local == "rPr" and not closing:
            if selfclose:
                run_rpr = m.group(0)
            else:
                in_rpr = True
                rpr_start = m.start()
            continue
        if local == "rPr" and closing:
            in_rpr = False
            run_rpr = xml[rpr_start:m.end()]
            continue
        if in_rpr:
            continue  # ignorar tudo dentro do rPr

        # --- limites de run e paragrafo ---
        if local == "r" and not closing:
            run_rpr = None
            continue
        if local == "p" and closing:
            flush()
            continue
        if local in ("tc", "tr") and closing:
            flush()
            continue

        # --- elementos que quebram o segmento ---
        if local in FLUSH_TAGS:
            flush()
            continue

        # --- <w:t> ---
        if local == "t" and not closing and not selfclose:
            attrs = m.group(4)
            pending_t = (m.end(), m.start(), m.end(), b"preserve" in attrs)
            continue
        if local == "t" and closing and pending_t is not None:
            content_start, tag_start, tag_end, preserve = pending_t
            raw = xml[content_start:m.start()]
            text = html.unescape(raw.decode("utf-8"))
            pending_t = None
            if cur_runs and cur_key != run_rpr:
                flush()
            cur_key = run_rpr
            cur_runs.append((content_start, m.start(), tag_start, tag_end, text))
            continue

    flush()
    return segments


def text_parts(zf: zipfile.ZipFile):
    """Partes XML do pacote que podem conter texto de corpo."""
    keep = []
    for n in zf.namelist():
        if not n.startswith("word/") or not n.endswith(".xml"):
            continue
        base = n.rsplit("/", 1)[-1]
        if base.startswith(("document", "header", "footer", "footnotes",
                            "endnotes", "comments")):
            keep.append(n)
    return sorted(keep)


NEEDS_TRANSLATION = re.compile(r"[A-Za-zÀ-ÿ]{2,}")


def build(docx_path):
    zf = zipfile.ZipFile(docx_path)
    index = {}
    for part in text_parts(zf):
        xml = zf.read(part)
        segs = segment_part(xml)
        if segs:
            index[part] = segs
    zf.close()
    return index


if __name__ == "__main__":
    docx = sys.argv[1]
    outdir = sys.argv[2]
    index = build(docx)

    # Deduplicar por texto exato para reduzir volume de traducao
    uniq = {}
    total_segs = 0
    skipped = 0
    for part, segs in index.items():
        for s in segs:
            total_segs += 1
            t = s["text"]
            if not NEEDS_TRANSLATION.search(t):
                skipped += 1
                continue
            uniq.setdefault(t, len(uniq))

    with open(f"{outdir}/segments.json", "w") as fh:
        json.dump({"order": [t for t, _ in sorted(uniq.items(), key=lambda kv: kv[1])]}, fh, ensure_ascii=False)

    # Mapa de posicoes para reinsercao posterior
    layout = {p: [{"runs": s["runs"], "text": s["text"]} for s in segs] for p, segs in index.items()}
    with open(f"{outdir}/layout.json", "w") as fh:
        json.dump(layout, fh, ensure_ascii=False)

    print(f"partes: {len(index)}")
    print(f"segmentos totais: {total_segs}")
    print(f"segmentos sem letras (nao traduzidos): {skipped}")
    print(f"segmentos unicos a traduzir: {len(uniq)}")
    print(f"caracteres unicos: {sum(len(t) for t in uniq)}")
