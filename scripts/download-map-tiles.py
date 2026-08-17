#!/usr/bin/env python3
"""从 palworld.gg 下载 Palworld 地图瓦片 (z0-z6) 并转为 webp。

零第三方依赖（转换用 Pillow）：
    pip install Pillow

用法：
    python scripts/download-map-tiles.py                  # 下载主世界
    python scripts/download-map-tiles.py --map feybreak   # 下载天坠之地
    python scripts/download-map-tiles.py --all            # 两张图都下
    python scripts/download-map-tiles.py --redown         # 强制重新下载
    python scripts/download-map-tiles.py --no-webp        # 只下 png 不转换
"""
import argparse
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from urllib.request import Request, urlopen

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# 两张地图：palpagos 主世界、feybreak 天坠之地（1.0 新地图）
MAPS = {
    "palpagos": {
        "url": "https://palworld.gg/images/tiles/{z}/{x}/{y}.png",
        "out": os.path.join(ROOT, "web", "public", "map", "tiles"),
        "tmp": os.path.join(ROOT, ".tmp", "map_tiles_png"),
        "referer": "https://palworld.gg/map",
    },
    "feybreak": {
        "url": "https://palworld.gg/images/world-tree-tiles/{z}/{x}/{y}.png",
        "out": os.path.join(ROOT, "web", "public", "map", "tiles-feybreak"),
        "tmp": os.path.join(ROOT, ".tmp", "map_tiles_feybreak"),
        "referer": "https://palworld.gg/map/world-tree",
    },
}

Z_TO_RANGE = {
    0: (0, 0),
    1: (1, 1),
    2: (3, 3),
    3: (7, 7),
    4: (15, 15),
    5: (31, 31),
    6: (63, 63),
}

UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"


def download_one(cfg, z, x, y, redown):
    png_path = os.path.join(cfg["tmp"], str(z), str(x), f"{y}.png")
    if os.path.exists(png_path) and not redown:
        return True, z, x, y, "cached"
    os.makedirs(os.path.dirname(png_path), exist_ok=True)
    url = cfg["url"].format(z=z, x=x, y=y)
    headers = {"User-Agent": UA, "Referer": cfg["referer"]}
    for attempt in range(3):
        try:
            req = Request(url, headers=headers)
            with urlopen(req, timeout=30) as resp:
                data = resp.read()
            with open(png_path, "wb") as f:
                f.write(data)
            return True, z, x, y, "downloaded"
        except Exception as e:
            code = getattr(getattr(e, "code", None), "__int__", lambda: 0)()
            if code in (403, 404):
                return False, z, x, y, f"skip({code})"
            if attempt == 2:
                return False, z, x, y, f"err({e})"
    return False, z, x, y, "failed"


def convert_to_webp(cfg):
    try:
        from PIL import Image
    except ImportError:
        print("Pillow 未安装，跳过 webp 转换。pip install Pillow")
        return
    count = 0
    for z in Z_TO_RANGE:
        z_dir = os.path.join(cfg["tmp"], str(z))
        if not os.path.isdir(z_dir):
            continue
        for x in os.listdir(z_dir):
            x_dir = os.path.join(z_dir, x)
            if not os.path.isdir(x_dir):
                continue
            for fn in os.listdir(x_dir):
                if not fn.endswith(".png"):
                    continue
                y = fn[:-4]
                out_dir = os.path.join(cfg["out"], str(z), x)
                out_path = os.path.join(out_dir, f"{y}.webp")
                os.makedirs(out_dir, exist_ok=True)
                try:
                    with Image.open(os.path.join(x_dir, fn)) as im:
                        im.save(out_path, "WEBP", quality=80, method=6)
                    count += 1
                except Exception as e:
                    print(f"转换失败 {z}/{x}/{y}: {e}")
    print(f"已转换 {count} 张 webp -> {cfg['out']}")


def process_map(name, args):
    cfg = MAPS[name]
    print(f"\n=== [{name}] {cfg['url']} ===")
    tasks = []
    for z, (xmax, ymax) in Z_TO_RANGE.items():
        for x in range(xmax + 1):
            for y in range(ymax + 1):
                tasks.append((z, x, y))
    total = len(tasks)
    print(f"共 {total} 个瓦片，开始下载（并发 16）...")

    done = 0
    skipped = 0
    failed = 0
    with ThreadPoolExecutor(max_workers=16) as ex:
        futs = [ex.submit(download_one, cfg, z, x, y, args.redown) for z, x, y in tasks]
        for fut in as_completed(futs):
            ok, z, x, y, msg = fut.result()
            done += 1
            if ok:
                pass
            elif msg.startswith("skip"):
                skipped += 1
            else:
                failed += 1
                print(f"[{done}/{total}] {z}/{x}/{y} {msg}")
            if done % 500 == 0:
                print(f"进度 {done}/{total}（跳过 {skipped}，失败 {failed}）")

    print(f"[{name}] 下载完成：{total - skipped - failed} 成功，{skipped} 跳过，{failed} 失败")
    if not args.no_webp:
        convert_to_webp(cfg)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--map", choices=list(MAPS.keys()), default="palpagos",
                        help="下载哪张地图（默认 palpagos 主世界）")
    parser.add_argument("--all", action="store_true", help="下载全部地图")
    parser.add_argument("--redown", action="store_true", help="强制重新下载")
    parser.add_argument("--no-webp", action="store_true", help="不转换 webp")
    args = parser.parse_args()

    names = list(MAPS.keys()) if args.all else [args.map]
    for name in names:
        process_map(name, args)


if __name__ == "__main__":
    main()
