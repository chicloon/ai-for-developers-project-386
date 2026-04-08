using Http;
using Rest;

namespace CallBooking;

@route("/api/my/bookings")
interface Bookings {
  @get list(): BookingsListResponse | ErrorResponse;

  @post create(@body booking: CreateBookingRequest): { @statusCode _: 201; ...Booking } | ErrorResponse;

  @delete
  @route("/{id}")
  cancel(@path id: string): { @statusCode _: 204; } | ErrorResponse;
}
