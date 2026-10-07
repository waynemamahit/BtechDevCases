export function Welcome({ email }: { email: string }) {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center bg-base-200 px-4 py-6 sm:px-6 md:px-8 lg:px-10">
      <div className="card w-full max-w-sm bg-base-100 shadow-sm sm:max-w-md md:max-w-lg">
        <div className="card-body">
          <p className="min-w-0 wrap-break-word">Hello {email}, welcome back</p>
        </div>
      </div>
    </main>
  );
}
