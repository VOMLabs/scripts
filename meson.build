project('headset-fixer', 'c',
  default_options: ['bindir=.'],)

go = find_program('go')

custom_target('headset-fixer',
  output: 'headset-fixer',
  input: 'main.go',
  build_by_default: true,
  console: true,
  command: [go, 'build', '-o', '@OUTPUT@', meson.current_source_dir()],
  install: false)
