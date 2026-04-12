## [1.0.10] - 2023-11-XX

### New Features
- **Add** GitWall cloud sync setup flow with store selection and token validation (`9b84309f`)

### Refactoring / Internal
- **Refactor** GitWall cloud setup to list and select existing stores instead of creating new ones (`9b84309f`)
- **Remove** automatic store creation in favor of user-managed stores via GitWall UI (`9b84309f`)
- **Update** cloud setup instructions to reference GitWall store management UI (`9b84309f`)

### Bug Fixes
- **Fix** edge case where `.envault` file in home directory could conflict with vault directory (`d5aced03`)
- **Improve** `.envault` file detection to stop at home directory boundary (`d5aced03`)
- **Add** error handling for `ENOTDIR` syscall when opening vault (`d5aced03`)

### Documentation
- **Add** comprehensive GitWall cloud sync documentation including setup, security, and usage (`c3868c8b`)
- **Update** installation script references to use GitWall-hosted URLs (`c3868c8b`)
- **Add** cloud sync commands to command reference table (`c3868c8b`)
- **Document** automatic sync behavior and multi-vault cloud config structure (`c3868c8b`)

## [1.0.9] - 2023-11-XX

### Performance
- **Improve** installation script by moving version fetch after architecture detection (`d5aced03`)

## [1.0.7] - 2023-11-XX

### Refactoring / Internal
- **Update** release workflow to skip deployment if tag already exists (`00c35107`)