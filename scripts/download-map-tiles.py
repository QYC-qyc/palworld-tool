#!/usr/bin/env python3
"""从 palworld.gg 下载 Palworld 地图瓦片 (z0-z6) 并转为 webp。

零第三方依赖（转换用 Pillow）：
    pip install Pillow

用法：
    python scripts/download-map-tiles.py            # 下载缺失瓦片并转 webp
    python scripts/download-map-tiles.py --redown   # 强制重新下载
    python scripts/download-map-tiles.py --no-webp  # 只下 png 不转换
"""
import argparse
import io
import os
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from urllib.request import Request, urlopen
from urllib.error import HTTPError

BASE_URL = "https://palworld.gg/images/tiles/{z}/{x}/{y}.png"
# 脚本位于 scripts/，瓦片输出到 web/public/map/tiles
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TILES_DIR = os.path.join(ROOT, "web", "public", "map", "tiles")
TMP_DIR = os.path.join(ROOT, ".tmp", "map_tiles_png")

Z_TO_RANGE = {
    0: (0, 0),
    1: (1, 1),
    2: (3, 3),
    3: (7, 7),
    4: (15, 15),
    5: (31, 31),
    6: (63, 63),
}

HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/120.0 Safari/537.36",
    "Referer": "https://palworld.gg/map",
}


def download_one(z, x, y, redown):
    png_path = os.path.join(TMP_DIR, str(z), str(x), f"{y}.png")
    if os.path.exists(png_path) and not redown:
        return True, z, x, y, "cached"
    os.makedirs(os.path.dirname(png_path), exist_ok=True)
    url = BASE_URL.format(z=z, x=x, y=y)
    for attempt in range(3):
        try:
            req = Request(url, headers=HEADERS)
            with urlopen(req, timeout=30) as resp:
                data = resp.read()
            with open(png_path, "wb") as f:
                f.write(data)
            return True, z, x, y, "downloaded"
        except HTTPError as e:
            if e.code in (403, 404):
                return False, z, x, y, f"skip({e.code})"
            if attempt == 2:
                return False, z, x, y, f"http({e.code})"
        except Exception as e:
            if attempt == 2:
                return False, z, x, y, f"err({e})"
    return False, z, x, y, "failed"


def convert_to_webp():
    try:
        from PIL import Image
    except ImportError:
        print("Pillow 未安装，跳过 webp 转换。pip install Pillow")
        return
    count = 0
    for z in Z_TO_RANGE:
        z_dir = os.path.join(TMP_DIR, str(z))
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
                out_dir = os.path.join(TILES_DIR, str(z), x)
                out_path = os.path.join(out_dir, f"{y}.webp")
                os.makedirs(out_dir, exist_ok=True)
                try:
                    with Image.open(os.path.join(x_dir, fn)) as im:
                        im.save(out_path, "WEBP", quality=80, method=6)
                    count += 1
                except Exception as e:
                    print(f"转换失败 {z}/{x}/{y}: {e}")
    print(f"已转换 {count} 张 webp -> {TILES_DIR}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--redown", action="store_true", help="强制重新下载")
    parser.add_argument("--no-webp", action="store_true", help="不转换 webp")
    args = parser.parse_args()

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
        futs = [ex.submit(download_one, z, x, y, args.redown) for z, x, y in tasks]
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
            if done % 200 == 0:
                print(f"进度 {done}/{total}（跳过 {skipped}，失败 {failed}）")

    print(f"下载完成：{total - skipped - failed} 成功，{skipped} 跳过，{failed} 失败")
    if not args.no_webp:
        convert_to_webp()


if __name__ == "__main__":
    main()
