#!/usr/bin/env python3
"""Render docs/FINAL_REPORT.ru.md to a PDF styled like a Markdown viewer."""
from __future__ import annotations

import re
from pathlib import Path

import markdown
from weasyprint import HTML, CSS

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "docs" / "FINAL_REPORT.ru.md"
OUT = ROOT / "docs" / "FINAL_REPORT.ru.pdf"


CSS_STYLES = """
@page {
  size: A4;
  margin: 22mm 18mm 22mm 18mm;
  @bottom-center {
    content: counter(page) " / " counter(pages);
    font-family: 'DejaVu Sans', 'Liberation Sans', sans-serif;
    font-size: 9pt;
    color: #6b7280;
  }
}

html { font-size: 11pt; }
body {
  font-family: 'DejaVu Sans', 'Liberation Sans', system-ui, -apple-system, sans-serif;
  color: #1f2328;
  line-height: 1.55;
  margin: 0;
}

h1, h2, h3, h4, h5, h6 {
  color: #1f2328;
  margin: 1.4em 0 0.55em;
  break-after: avoid-page;
  page-break-after: avoid;
  break-inside: avoid;
  line-height: 1.25;
  font-weight: 600;
}
h1 {
  font-size: 1.9em;
  border-bottom: 1px solid #d0d7de;
  padding-bottom: 0.3em;
}
h2 {
  font-size: 1.45em;
  border-bottom: 1px solid #d0d7de;
  padding-bottom: 0.25em;
}
h3 { font-size: 1.18em; }
h4 { font-size: 1.0em; }

p { margin: 0.6em 0; }
ul, ol { margin: 0.5em 0 0.8em 1.25em; padding: 0; }
li { margin: 0.18em 0; }
li > p { margin: 0.2em 0; }

a { color: #0969da; text-decoration: none; }

strong { font-weight: 600; }

code, pre, tt {
  font-family: 'DejaVu Sans Mono', 'Liberation Mono', 'Menlo', 'Consolas', monospace;
}
code, tt { font-size: 0.88em; }
pre { font-size: 0.74em; }
p code, li code, td code, th code {
  background: #f5f6f8;
  border: 1px solid #e1e4e8;
  border-radius: 4px;
  padding: 0.05em 0.35em;
  color: #24292f;
  white-space: nowrap;
}
pre {
  background: #f6f8fa;
  border: 1px solid #e1e4e8;
  border-radius: 6px;
  padding: 0.7em 0.9em;
  line-height: 1.45;
  white-space: pre;
  overflow-x: hidden;
}
pre code {
  background: transparent;
  border: 0;
  padding: 0;
  font-size: inherit;
  color: #24292f;
  white-space: pre;
}

blockquote {
  margin: 0.8em 0;
  padding: 0.05em 1em;
  color: #57606a;
  border-left: 4px solid #d0d7de;
  background: #f6f8fa;
}
blockquote p { margin: 0.5em 0; }

table {
  border-collapse: collapse;
  margin: 0.7em 0 1em;
  width: 100%;
  font-size: 0.92em;
}
thead { display: table-header-group; }
tr { break-inside: avoid; page-break-inside: avoid; }
th, td {
  border: 1px solid #d0d7de;
  padding: 0.4em 0.65em;
  text-align: left;
  vertical-align: top;
}
thead th {
  background: #f6f8fa;
  font-weight: 600;
}
tbody tr:nth-child(even) td {
  background: #fbfcfd;
}

hr {
  border: none;
  border-top: 1px solid #d0d7de;
  margin: 1.4em 0;
}

.keep-together {
  break-before: page;
  page-break-before: always;
  break-inside: avoid;
  page-break-inside: avoid;
}

/* Pygments code highlighting (codehilite) — minimal palette */
.codehilite { background: #f6f8fa; border-radius: 6px; }
.codehilite .k, .codehilite .kd, .codehilite .kn, .codehilite .kr { color: #cf222e; }   /* keywords */
.codehilite .s, .codehilite .s1, .codehilite .s2 { color: #0a3069; }                    /* strings */
.codehilite .c, .codehilite .c1, .codehilite .cm { color: #6e7781; font-style: italic; }/* comments */
.codehilite .nf, .codehilite .nx { color: #8250df; }                                    /* function names */
.codehilite .mi, .codehilite .mf { color: #0550ae; }                                    /* numbers */
.codehilite .o { color: #cf222e; }                                                      /* operators */
.codehilite .nt { color: #116329; }                                                     /* tags */
"""


def main() -> None:
    md_text = SRC.read_text(encoding="utf-8")
    html_body = markdown.markdown(
        md_text,
        extensions=[
            "extra",            # tables, fenced_code, attr_list, def_list, footnotes
            "sane_lists",
            "codehilite",       # Pygments highlighting
            "toc",
        ],
        extension_configs={
            "codehilite": {"guess_lang": False, "css_class": "codehilite"},
        },
        output_format="html5",
    )
    # Keep §17 (file tree section) on a single page.
    html_body = re.sub(
        r'(<h2[^>]*>\s*17\.[^<]+</h2>.*?)(?=<h2|<hr)',
        r'<div class="keep-together">\1</div>',
        html_body,
        count=1,
        flags=re.DOTALL,
    )

    full_html = f"""<!doctype html>
<html lang="ru">
<head><meta charset="utf-8"><title>Pollify — финальный отчёт</title></head>
<body>{html_body}</body>
</html>
"""
    HTML(string=full_html, base_url=str(ROOT)).write_pdf(
        target=str(OUT),
        stylesheets=[CSS(string=CSS_STYLES)],
    )
    print(f"wrote {OUT}")


if __name__ == "__main__":
    main()
