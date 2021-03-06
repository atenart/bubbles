// Copyright (C) 2019 Antoine Tenart <antoine.tenart@ack.tf>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//

$(function() {
  // Retrieve the anchor to display.
  var anchor = window.top.location.hash.substr(1);

  // Check the anchor exists.
  if (anchor.length == 0 || $("#modal-" + anchor).length == 0)
    return;

  // Display the anchor.
  $(`#modal-${anchor}`).addClass("is-active");
})

function showModal(id) {
  $(`#modal-${id} form`)[0].reset();
  $(`#modal-${id}`).addClass("is-active");
}

function hideModal(id) {
  $(`#modal-${id}`).removeClass("is-active");
}
