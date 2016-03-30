# Change Log
All notable changes to this project will be documented in this file. See [Keep a
CHANGELOG](http://keepachangelog.com/) for how to update this file. This project
adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased][unreleased]

## [0.0.2] - 2016-03-28
### Added
- Make newError function public (#6). -@kostyantyn
- Add public access to default client (#5). -@kostyantyn
- Support default server mux in Handler.
- Allow error class to be customized from `honeybadger.Notify`.
- Support sending fingerprint in `honeybadger.Notify`.
- Added BeforeNotify callback.

### Fixed
- Drain the body of a response before closing it (#4). -@kostyantyn
- Update config at pointer rather than dereferencing. (#2).

## [0.0.1] - 2015-06-25
### Added
- Go client for Honeybadger.io.
