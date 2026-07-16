export function CommandCenterSkeleton() {
  return (
    <section className="rounded-[20px] border border-sand-300 bg-white p-6">
      <div className="h-4 w-36 rounded-full bg-sand-200" />
      <div className="mt-4 h-8 w-2/3 rounded-full bg-sand-200" />
      <div className="mt-3 h-4 w-full max-w-[620px] rounded-full bg-sand-200" />
      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <div className="h-44 rounded-[18px] bg-sand-100" />
        <div className="h-44 rounded-[18px] bg-sand-100" />
      </div>
    </section>
  );
}
