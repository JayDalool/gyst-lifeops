// Presentation-only period switching. No persistence or network requests.
(function () {
  var controls = document.querySelector('.health-period');
  if (!controls) return;
  var daily = document.querySelector('[data-period-daily]');
  var history = document.querySelector('[data-period-history]');
  var descriptions = {
    Week: 'Average calories, nutrition target adherence, workouts completed, average steps, and weight trend.',
    Month: 'Weight trend, nutrition averages, workouts, activity, and strength trend.',
    Year: 'Long-term body weight, strength progression, training sessions, and average activity.'
  };
  controls.hidden = false;
  controls.addEventListener('click', function (event) {
    var button = event.target.closest('[data-period]');
    if (!button) return;
    var period = button.dataset.period;
    controls.querySelectorAll('button').forEach(function (item) { item.setAttribute('aria-pressed', String(item === button)); });
    daily.hidden = period !== 'Day';
    history.hidden = period === 'Day';
    history.querySelector('[data-period-title]').textContent = period + ' summary';
    history.querySelector('[data-period-description]').textContent = descriptions[period] || '';
  });
  document.querySelectorAll('.health-links a').forEach(function (link) {
    link.addEventListener('click', function () {
      if (link.hash === '#nutrition' || link.hash === '#training') controls.querySelector('[data-period="Day"]').click();
    });
  });
})();
