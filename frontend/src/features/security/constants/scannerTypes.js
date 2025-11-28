export const SCANNER_TYPES = {
  SYFT_GRYPE: 'Syft/Grype',
  OWASP_BLINT: 'OWASP BlInt',
  OWASP_DEPSCAN: 'OWASP Dep-Scan'
};

export const SCANNER_DESCRIPTIONS = {
  [SCANNER_TYPES.SYFT_GRYPE]: 'SBOM generation and vulnerability scanning',
  [SCANNER_TYPES.OWASP_BLINT]: 'Binary analysis and vulnerability detection',
  [SCANNER_TYPES.OWASP_DEPSCAN]: 'Dependency vulnerability scanning'
};