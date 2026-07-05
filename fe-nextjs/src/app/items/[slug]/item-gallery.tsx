"use client";

import { ChevronLeft, ChevronRight, Maximize2, X } from "lucide-react";
import Image from "next/image";
import type { MouseEvent } from "react";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { useT } from "@/i18n";

interface ItemGalleryProps {
  photos: string[];
  title: string;
}

function replacePhotoTokens(template: string, current: number, total: number) {
  return template.replace("{current}", String(current)).replace("{total}", String(total));
}

export function ItemGallery({ photos, title }: ItemGalleryProps) {
  const t = useT();
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [pointerStartX, setPointerStartX] = useState<number | null>(null);
  const activePhoto = activeIndex === null ? null : photos[activeIndex];
  const total = photos.length;
  const heroPhoto = photos[0];
  const hasMultiplePhotos = total > 1;

  const showPrevious = useCallback(() => {
    setActiveIndex((current) => {
      if (current === null) return current;
      return (current - 1 + total) % total;
    });
  }, [total]);

  const showNext = useCallback(() => {
    setActiveIndex((current) => {
      if (current === null) return current;
      return (current + 1) % total;
    });
  }, [total]);

  const closeViewer = useCallback(() => {
    setActiveIndex(null);
    setPointerStartX(null);
  }, []);

  useEffect(() => {
    if (activeIndex === null) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") closeViewer();
      if (event.key === "ArrowLeft" && hasMultiplePhotos) showPrevious();
      if (event.key === "ArrowRight" && hasMultiplePhotos) showNext();
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [activeIndex, closeViewer, hasMultiplePhotos, showNext, showPrevious]);

  const photoCounter = (index: number) =>
    replacePhotoTokens(t.listing.photoCounter, index + 1, total);
  const openPhotoLabel = (index: number) =>
    t.listing.openPhoto.replace("{number}", String(index + 1));
  const handlePointerUp = (clientX: number) => {
    if (!hasMultiplePhotos || pointerStartX === null) return;

    const deltaX = clientX - pointerStartX;
    if (Math.abs(deltaX) > 44) {
      if (deltaX > 0) showPrevious();
      if (deltaX < 0) showNext();
    }
    setPointerStartX(null);
  };
  const handlePreviousClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    showPrevious();
  };
  const handleNextClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    showNext();
  };

  return (
    <>
      <div className="space-y-3">
        <button
          type="button"
          onClick={() => setActiveIndex(0)}
          className="group relative block aspect-[4/3] w-full overflow-hidden rounded-[16px] bg-surface-soft text-left shadow-card"
          aria-label={openPhotoLabel(0)}
        >
          <Image
            src={heroPhoto}
            alt={title}
            fill
            priority
            className="object-cover transition-transform duration-300 group-hover:scale-[1.015]"
            sizes="(max-width: 1024px) 100vw, 58vw"
          />
          <span className="absolute bottom-4 right-4 inline-flex size-10 items-center justify-center rounded-full bg-white/92 text-text-primary shadow-[0_8px_20px_rgba(32,28,64,0.18)] transition-transform group-hover:scale-105">
            <Maximize2 className="size-4" />
          </span>
        </button>

        {hasMultiplePhotos && (
          <div className="grid grid-cols-4 gap-2 sm:grid-cols-6 lg:grid-cols-8">
            {photos.map((photo, index) => (
              <button
                type="button"
                key={photo}
                onClick={() => setActiveIndex(index)}
                className="relative aspect-square overflow-hidden rounded-lg border border-transparent bg-surface-soft transition-all hover:border-primary focus-visible:border-primary focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-primary/20"
                aria-label={openPhotoLabel(index)}
              >
                <Image
                  src={photo}
                  alt={`${title} - ${photoCounter(index)}`}
                  fill
                  className="object-cover"
                  sizes="96px"
                />
              </button>
            ))}
          </div>
        )}
      </div>

      {activePhoto && activeIndex !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 px-3 py-5">
          <button
            type="button"
            className="absolute inset-0"
            aria-label={t.listing.closePhoto}
            onClick={closeViewer}
          />

          <div
            className="relative z-10 flex h-full w-full max-w-6xl flex-col"
            role="dialog"
            aria-modal="true"
            aria-label={photoCounter(activeIndex)}
          >
            <div className="mb-3 flex items-center justify-between gap-3 text-white">
              <p className="text-sm font-semibold">{photoCounter(activeIndex)}</p>
              <Button
                type="button"
                variant="ghost"
                size="icon-lg"
                aria-label={t.listing.closePhoto}
                className="z-30 bg-white/10 text-white hover:bg-white/20 hover:text-white"
                onClick={closeViewer}
              >
                <X className="size-5" />
              </Button>
            </div>

            <div className="relative min-h-0 flex-1">
              <div
                className="absolute inset-0 touch-pan-y"
                onPointerDown={(event) => setPointerStartX(event.clientX)}
                onPointerUp={(event) => handlePointerUp(event.clientX)}
                onPointerCancel={() => setPointerStartX(null)}
              >
                <Image
                  src={activePhoto}
                  alt={`${title} - ${photoCounter(activeIndex)}`}
                  fill
                  className="pointer-events-none object-contain"
                  sizes="100vw"
                  priority
                />
              </div>

              {hasMultiplePhotos && (
                <>
                  <button
                    type="button"
                    aria-label={t.listing.previousPhoto}
                    className="absolute left-2 top-1/2 z-30 flex size-12 -translate-y-1/2 items-center justify-center rounded-full bg-white text-text-primary shadow-[0_12px_34px_rgba(0,0,0,0.34)] transition-transform hover:scale-105 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-white/60 sm:left-5 sm:size-14"
                    onClick={handlePreviousClick}
                  >
                    <ChevronLeft className="size-6" />
                  </button>
                  <button
                    type="button"
                    aria-label={t.listing.nextPhoto}
                    className="absolute right-2 top-1/2 z-30 flex size-12 -translate-y-1/2 items-center justify-center rounded-full bg-white text-text-primary shadow-[0_12px_34px_rgba(0,0,0,0.34)] transition-transform hover:scale-105 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-white/60 sm:right-5 sm:size-14"
                    onClick={handleNextClick}
                  >
                    <ChevronRight className="size-6" />
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
