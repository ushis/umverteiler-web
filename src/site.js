(function() {
  'use strict';

  document.addEventListener('click', function(e) {
    if (e.target.matches('[data-toggle-class]')) {
      var target = document.querySelector(e.target.dataset.toggleClassTarget);
      var klass = e.target.dataset.toggleClass;
      target.classList.toggle(klass);
      e.preventDefault();
      e.stopPropagation();
      return;
    }
  });

  document.addEventListener('DOMContentLoaded', function() {
    setupBalanceMeter();
  });

  var setupBalanceMeter = function() {
    var container = document.querySelector('[data-balance-meter]');
    var url = container.dataset.balanceMeter;
    var xhr = new XMLHttpRequest();

    xhr.addEventListener('load', function(e) {
      var current = container.querySelector('.current');
      var progress = container.querySelector('.progress');
      var period = container.querySelector('.period');

      var balance = xhr.response.balance;
      var costs = Number.parseFloat(period.dataset.fixedCosts);
      var target = progress.max;
      var ratio = balance / target;

      if (ratio >= 0.5) {
        progress.classList.add('is-success');
      } else if (ratio >= 0.25) {
        progress.classList.add('is-warning');
      } else {
        progress.classList.add('is-danger');
      }

      var t = 0.0;

      var interval = window.setInterval(function() {
        var value = Math.floor(balance * t * (2 - t));
        current.textContent = value;
        progress.value = value;
        period.textContent = Math.floor(value / costs);

        if (t >= 1.0) {
          window.clearInterval(interval);
        } else {
          t += 0.005;
        }
      }, 10);
    });

    xhr.responseType = 'json';
    xhr.open('get', url);
    xhr.send();
  };
}).call(this);
