import { darkTheme, type GlobalThemeOverrides } from 'naive-ui'

export { darkTheme }

// 主色：现代靛蓝
export const PRIMARY_COLOR = '#4f46e5'
export const PRIMARY_COLOR_HOVER = '#6366f1'
export const PRIMARY_COLOR_PRESSED = '#4338ca'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: PRIMARY_COLOR,
    primaryColorHover: PRIMARY_COLOR_HOVER,
    primaryColorPressed: PRIMARY_COLOR_PRESSED,
    primaryColorSuppl: PRIMARY_COLOR_HOVER,
    borderRadius: '10px',
    borderRadiusSmall: '6px',
    fontFamily:
      '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", "Helvetica Neue", Helvetica, Arial, sans-serif',
    fontWeightStrong: '600',
  },
  Card: {
    borderRadius: '12px',
    paddingMedium: '18px 20px',
  },
  Button: {
    borderRadiusMedium: '8px',
    borderRadiusSmall: '6px',
  },
  Tag: {
    borderRadius: '6px',
  },
  Menu: {
    itemBorderRadius: '8px',
  },
}
