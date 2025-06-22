<script lang="ts">
  import { onMount } from 'svelte';
  import L from 'leaflet';
  import Chart from 'chart.js/auto';

  let mapContainer;
  let map;

  // Palette (fill, outline)
  const bixiRed = ['#ee3124', '#931910'];
  const ebixiBlue = ['#007ecc', '#005e99'];
  const disabledGrey = ['#a0a0a0', '#878787'];

  function createChartPopup(stationId, stationName) {
    const container = document.createElement('div');
    container.innerHTML = `
      <div><strong>${stationName}</strong></div>
      <div class="current-availability">
        <div class="availability-item bikes-classic">
          <span class="availability-count">-</span> Classique
        </div>
        <div class="availability-item bikes-electric">
          <span class="availability-count">-</span> Électrique
        </div>
      </div>
      <div class='chart-container'>
        <canvas id='chart-${stationId}' width='300' height='200'></canvas>
      </div>
    `;
    
    fetch(`/stations/${stationId}`)
      .then(res => res.json())
      .then(data => {
        // Update current availability display
        const classicCount = container.querySelector('.bikes-classic .availability-count');
        const electricCount = container.querySelector('.bikes-electric .availability-count');
        
        classicCount.textContent = Math.floor(data.current.b).toString();
        electricCount.textContent = Math.floor(data.current.eb).toString();
        
        const ctx = (container.querySelector(`#chart-${stationId}`) as HTMLCanvasElement).getContext('2d');
        const labels = data.historical.map(row => {
          const d = new Date(row.t * 1000);
          return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
        });
        const bikesAvailable = data.historical.map(row => row.b);
        const ebikesAvailable = data.historical.map(row => row.eb);

        function makeGradient(ctx, color) {
          const gradient = ctx.createLinearGradient(0, 0, 0, 200);
          gradient.addColorStop(0, color + 'CC'); // near line, more opaque
          gradient.addColorStop(1, color + '00'); // bottom, transparent
          return gradient;
        }

        new Chart(ctx, {
          type: 'line',
          data: {
            labels: labels,
            datasets: [
              {
                label: 'Classique',
                data: bikesAvailable,
                borderColor: bixiRed[0],
                backgroundColor: makeGradient(ctx, bixiRed[0]),
                fill: true,
                pointRadius: 0,
                pointHoverRadius: 0
              },
              {
                label: 'Électrique',
                data: ebikesAvailable,
                borderColor: ebixiBlue[0],
                backgroundColor: makeGradient(ctx, ebixiBlue[0]),
                fill: true,
                pointRadius: 0,
                pointHoverRadius: 0
              }
            ]
          },
          options: {
            responsive: false,
            plugins: { legend: { display: true } },
            scales: {
              x: {
                display: true,
                ticks: {
                  callback: function(val, idx) { return labels[idx]; }
                }
              },
              y: {
                display: true,
                min: 0,
                max: data.capacity,
              }
            }
          }
        });
      })
      .catch(() => {
        container.innerHTML += '<div>Failed to load chart data.</div>';
      });
    return container;
  }

  onMount(() => {
    map = L.map(mapContainer).setView([0, 0], 2);
    L.tileLayer('/map/{z}/{x}/{y}', {
      maxZoom: 19,
      attribution: `© <a href="https://www.mapbox.com/about/maps">Mapbox</a> © <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a> <strong><a href="https://apps.mapbox.com/feedback/" target="_blank">Improve this map</a></strong>`
    }).addTo(map);

    fetch('/stations')
      .then(response => response.json())
      .then(geojson => {
        const geoLayer = L.geoJSON(geojson, {
          pointToLayer: function (feature, latlng) {
            return L.circleMarker(latlng, {
              radius: 6,
              fillColor: feature.properties.active ? bixiRed[0] : disabledGrey[0],
              color: feature.properties.active ? bixiRed[1] : disabledGrey[1],
              weight: 2,
              opacity: 1,
              fillOpacity: 1
            });
          },
          onEachFeature: function (feature, layer) {
            if (feature.properties && feature.properties.id && feature.properties.name) {
              layer.on('click', function() {
                const popupContent = createChartPopup(feature.properties.id, feature.properties.name);
                layer.bindPopup(popupContent).openPopup();
              });
            }
          }
        }).addTo(map);
        if (geoLayer.getBounds().isValid()) {
          map.fitBounds(geoLayer.getBounds());
        }
      })
      .catch(err => {
        alert('Failed to load stations: ' + err);
      });

    return () => {
      if (map) {
        map.remove();
      }
    };
  });
</script>

<div bind:this={mapContainer} id="map"></div>

<style>
  #map {
    height: 100vh;
    width: 100vw;
  }

  :global(.chart-container) {
    width: 300px;
    height: 200px;
  }

  :global(.current-availability) {
    padding: 10px 0;
    border-bottom: 1px solid #ddd;
    margin-bottom: 10px;
  }

  :global(.availability-item) {
    display: inline-block;
    margin-right: 15px;
    font-size: 14px;
  }

  :global(.availability-count) {
    font-weight: bold;
    font-size: 16px;
  }

  :global(.bikes-classic) {
    color: #ee3124;
  }

  :global(.bikes-electric) {
    color: #007ecc;
  }
</style>
