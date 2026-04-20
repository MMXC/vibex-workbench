#!/usr/bin/env python3
# Icon generator - generates icons from SVG
# Dependencies: pip install cairosvg Pillow

from pathlib import Path

ICONS_DIR = Path(__file__).parent / 'icons'
ICONS_DIR.mkdir(exist_ok=True)

SVG = '''
<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 512 512'>
  <rect width='512' height='512' rx='96' fill='#6366f1'/>
  <text x='256' y='340' text-anchor='middle' font-family='system-ui' font-size='280' font-weight='700' fill='white'>V</text>
</svg>
'''

def generate_icons():
    try:
        import cairosvg
        svg_file = ICONS_DIR / 'icon.svg'
        svg_file.write_text(SVG)
        sizes = [32, 128, 256, 512]
        for size in sizes:
            out = ICONS_DIR / f'{size}x{size}.png'
            cairosvg.svg2png(url=str(svg_file), write_to=str(out), output_width=size, output_height=size)
            print(f'  Generated {size}x{size}.png')
        out2x = ICONS_DIR / '128x128@2x.png'
        cairosvg.svg2png(url=str(svg_file), write_to=str(out2x), output_width=256, output_height=256)
        print('  Generated 128x128@2x.png')
    except ImportError:
        print('cairosvg not installed - skipping icon generation')

if __name__ == '__main__':
    generate_icons()
