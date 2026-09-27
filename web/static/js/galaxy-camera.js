// Fit a sphere around every project in the narrower camera dimension.
export function galaxyCameraFit(radius, width, height, verticalFOV = 55) {
  const aspect = Math.max(1, width) / Math.max(1, height);
  const vertical = verticalFOV * Math.PI / 360;
  const horizontal = Math.atan(Math.tan(vertical) * aspect);
  return Math.max(58, radius * 1.18 / Math.sin(Math.min(vertical, horizontal)));
}
