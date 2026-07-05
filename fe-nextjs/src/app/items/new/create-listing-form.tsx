"use client";

import { CheckCircle2, ImagePlus, Loader2, Sparkles, Trash2, UploadCloud } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import type React from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getCategories } from "@/features/catalog/get-categories";
import { getCities } from "@/features/catalog/get-cities";
import { type UploadedItemPhoto, uploadItemPhoto } from "@/features/listing/upload-item-photo";
import { type CreateItemRequest, useCreateItem } from "@/features/listing/use-create-item";
import { useUpdateItem } from "@/features/listing/use-update-item";
import { useT } from "@/i18n";
import type { Category, City, Item } from "@/lib/api/types";
import { isDemoMode } from "@/lib/demo";
import { cn } from "@/lib/utils";

interface CreateListingFormProps {
  locale: string;
  item?: Item;
  mode?: "create" | "edit";
}

type ListingCondition = "new" | "used";
type ListingPhoto = UploadedItemPhoto & { fileName?: string };

const maxPhotos = 10;

export function CreateListingForm({ locale, item, mode = "create" }: CreateListingFormProps) {
  const t = useT();
  const router = useRouter();
  const createMutation = useCreateItem();
  const updateMutation = useUpdateItem(item?.id ?? "");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const isEdit = mode === "edit" && !!item?.id;
  const activeMutation = isEdit ? updateMutation : createMutation;
  const priceCurrency = item?.price?.currency || defaultCurrencyForLocale(locale);

  const [title, setTitle] = useState(item?.title ?? "");
  const [categoryId, setCategoryId] = useState(item?.category_id ?? "");
  const [cityId, setCityId] = useState(item?.city?.id ?? "");
  const [condition, setCondition] = useState<ListingCondition>(
    item?.condition === "used" ? "used" : "new",
  );
  const [priceAmount, setPriceAmount] = useState(
    item?.price?.amount != null ? String(item.price.amount / 100) : "",
  );
  const [description, setDescription] = useState(item?.description ?? "");
  const [photos, setPhotos] = useState<ListingPhoto[]>(() =>
    (item?.photos ?? [])
      .filter((photo): photo is ListingPhoto => Boolean(photo.id && photo.url))
      .map((photo) => ({
        id: photo.id ?? "",
        url: photo.url ?? photo.thumbnail_url ?? "",
        thumbnail_url: photo.thumbnail_url ?? photo.url ?? "",
      })),
  );

  const [categories, setCategories] = useState<Category[]>([]);
  const [cities, setCities] = useState<City[]>([]);
  const [loading, setLoading] = useState(true);
  const [demoError, setDemoError] = useState("");
  const [formError, setFormError] = useState("");
  const [uploadingCount, setUploadingCount] = useState(0);

  useEffect(() => {
    Promise.all([getCategories(locale as "en" | "ru"), getCities()])
      .then(([cats, citys]) => {
        setCategories(cats);
        setCities(citys);
      })
      .catch(() => {
        setFormError(t.states.error);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [locale, t.states.error]);

  const selectedCategory = useMemo(
    () => categories.find((category) => category.id === categoryId),
    [categories, categoryId],
  );

  const selectedCity = useMemo(() => cities.find((city) => city.id === cityId), [cities, cityId]);

  const cityLabel =
    selectedCity?.names?.[locale as "en" | "ru"] || selectedCity?.names?.en || selectedCity?.slug;

  const isRequiredFieldsFilled = !!title.trim() && !!categoryId && !!cityId;
  const isSubmitDisabled =
    !isRequiredFieldsFilled || activeMutation.isPending || uploadingCount > 0;

  const handlePhotoSelect = async (files: FileList | null) => {
    if (!files || files.length === 0) return;

    setFormError("");
    setDemoError("");

    if (isDemoMode()) {
      setDemoError(t.demo.disabledAction);
      return;
    }

    const availableSlots = maxPhotos - photos.length;
    const selectedFiles = Array.from(files).slice(0, availableSlots);
    if (selectedFiles.length === 0) {
      setFormError(t.newListing.photos_limit);
      return;
    }

    setUploadingCount(selectedFiles.length);
    try {
      const uploaded = await Promise.all(
        selectedFiles.map(async (file) => {
          const photo = await uploadItemPhoto(file, isEdit ? item?.id : undefined);
          return { ...photo, fileName: file.name };
        }),
      );
      setPhotos((current) => [...current, ...uploaded].slice(0, maxPhotos));
    } catch (error) {
      setFormError(error instanceof Error ? error.message : t.newListing.photos_error);
    } finally {
      setUploadingCount(0);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  const removePhoto = (id: string) => {
    setPhotos((current) => current.filter((photo) => photo.id !== id));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    setFormError("");
    setDemoError("");

    if (!isRequiredFieldsFilled) {
      setFormError(t.newListing.error_required);
      return;
    }

    if (isDemoMode()) {
      setDemoError(t.demo.disabledAction);
      return;
    }

    const photoIds = photos.map((photo) => photo.id).filter(Boolean);
    const firstPhotoId = photoIds[0];

    const body: CreateItemRequest = {
      category_id: categoryId,
      title: title.trim(),
      description: description.trim() || undefined,
      condition,
      city_uuid: cityId,
      photo_ids: photoIds,
      thumbnail_id: firstPhotoId,
    };

    if (priceAmount.trim()) {
      const amount = Number.parseFloat(priceAmount);
      if (!Number.isNaN(amount) && amount >= 0) {
        body.price = {
          amount: Math.round(amount * 100),
          currency: priceCurrency,
        };
      }
    }

    if (isEdit) {
      updateMutation.mutate(
        {
          ...body,
          thumbnail_id: firstPhotoId ?? null,
        },
        {
          onSuccess: (data) => {
            router.push(`/items/${data.slug || data.id}`);
          },
          onError: (error) => {
            setFormError(error.message || t.newListing.error_generic);
          },
        },
      );
      return;
    }

    createMutation.mutate(body, {
      onSuccess: (data) => {
        router.push(`/items/${data.slug || data.id}`);
      },
      onError: (error) => {
        setFormError(error.message || t.newListing.error_generic);
      },
    });
  };

  if (loading) {
    return (
      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div className="h-[520px] animate-pulse rounded-lg bg-surface-soft" />
        <div className="h-[360px] animate-pulse rounded-lg bg-surface-soft" />
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
      <section className="min-w-0 rounded-lg border border-border bg-white/94 p-5 shadow-card md:p-6">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-3 border-b border-border pb-5">
          <div>
            <p className="text-meta font-semibold uppercase text-primary">
              {isEdit ? t.newListing.edit_eyebrow : t.newListing.form_eyebrow}
            </p>
            <h2 className="mt-1 text-h2 text-text-primary">
              {isEdit ? t.newListing.edit_title : t.newListing.details_title}
            </h2>
          </div>
          <div className="inline-flex items-center gap-2 rounded-lg bg-surface-soft px-3 py-2 text-meta font-medium text-text-secondary">
            <CheckCircle2 className="size-4 text-success" />
            {t.newListing.autosave_hint}
          </div>
        </div>

        {(demoError || formError) && (
          <div className="mb-5 rounded-lg border border-destructive/20 bg-destructive/10 px-4 py-3">
            <p className="text-sm font-medium text-destructive">{demoError || formError}</p>
          </div>
        )}

        <div className="grid gap-5 md:grid-cols-2">
          <Field
            label={t.newListing.title_label}
            htmlFor="title"
            required
            className="md:col-span-2"
          >
            <Input
              id="title"
              type="text"
              placeholder={t.newListing.title_placeholder}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={activeMutation.isPending}
              required
              maxLength={100}
              className="h-11 bg-white px-3 text-[15px]"
            />
          </Field>

          <Field label={t.newListing.category_label} htmlFor="category" required>
            <SelectField
              id="category"
              value={categoryId}
              onChange={setCategoryId}
              disabled={activeMutation.isPending}
              placeholder={t.newListing.category_placeholder}
              required
            >
              {categories.map((cat) => (
                <option key={cat.id} value={cat.id}>
                  {cat.name}
                </option>
              ))}
            </SelectField>
          </Field>

          <Field label={t.newListing.city_label} htmlFor="city" required>
            <SelectField
              id="city"
              value={cityId}
              onChange={setCityId}
              disabled={activeMutation.isPending}
              placeholder={t.newListing.city_placeholder}
              required
            >
              {cities.map((city) => (
                <option key={city.id} value={city.id}>
                  {city.names?.[locale as "en" | "ru"] || city.names?.en || city.slug}
                </option>
              ))}
            </SelectField>
          </Field>

          <Field label={t.newListing.condition_label}>
            <div className="grid grid-cols-2 gap-2 rounded-lg bg-surface-soft p-1">
              {(["new", "used"] as const).map((value) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setCondition(value)}
                  disabled={activeMutation.isPending}
                  className={cn(
                    "h-10 rounded-lg text-sm font-semibold transition-colors",
                    condition === value
                      ? "bg-white text-primary shadow-[0_6px_18px_rgba(44,42,84,0.08)]"
                      : "text-text-secondary hover:text-text-primary",
                  )}
                >
                  {value === "new" ? t.newListing.condition_new : t.newListing.condition_used}
                </button>
              ))}
            </div>
          </Field>

          <Field label={t.newListing.price_label} htmlFor="price">
            <div className="flex h-11 overflow-hidden rounded-lg border border-input bg-white focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50">
              <input
                id="price"
                type="number"
                placeholder={t.newListing.price_placeholder}
                value={priceAmount}
                onChange={(e) => setPriceAmount(e.target.value)}
                disabled={activeMutation.isPending}
                min="0"
                step="0.01"
                inputMode="decimal"
                className="min-w-0 flex-1 bg-transparent px-3 text-[15px] outline-none"
              />
              <div className="flex items-center border-l border-border px-3 text-sm font-semibold text-text-secondary">
                {priceCurrency}
              </div>
            </div>
          </Field>

          <Field
            label={t.newListing.description_label}
            htmlFor="description"
            className="md:col-span-2"
          >
            <textarea
              id="description"
              placeholder={t.newListing.description_placeholder}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={activeMutation.isPending}
              maxLength={2000}
              rows={7}
              className={cn(
                "w-full resize-none rounded-lg border border-input bg-white px-3 py-3 text-[15px] leading-6 outline-none transition-colors",
                "placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50",
                "disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50",
              )}
            />
          </Field>
        </div>
      </section>

      <aside className="min-w-0 space-y-6">
        <section className="rounded-lg border border-border bg-white/94 p-5 shadow-card">
          <div className="flex items-start gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <ImagePlus className="size-5" />
            </div>
            <div className="min-w-0">
              <h2 className="text-section text-text-primary">{t.newListing.photos_title}</h2>
              <p className="mt-1 text-body text-text-secondary">{t.newListing.photos_hint}</p>
            </div>
          </div>

          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={activeMutation.isPending || uploadingCount > 0 || photos.length >= maxPhotos}
            className={cn(
              "mt-5 flex min-h-[132px] w-full flex-col items-center justify-center rounded-lg border border-dashed border-primary/35 bg-primary/5 px-4 py-5 text-center transition-colors",
              "hover:border-primary hover:bg-primary/10 disabled:pointer-events-none disabled:opacity-60",
            )}
          >
            {uploadingCount > 0 ? (
              <Loader2 className="size-7 animate-spin text-primary" />
            ) : (
              <UploadCloud className="size-8 text-primary" />
            )}
            <span className="mt-3 text-sm font-bold text-text-primary">
              {uploadingCount > 0 ? t.newListing.photos_uploading : t.newListing.photos_drop}
            </span>
            <span className="mt-1 text-meta text-text-muted">{t.newListing.photos_format}</span>
          </button>

          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            multiple
            className="hidden"
            onChange={(event) => handlePhotoSelect(event.target.files)}
          />

          {photos.length > 0 && (
            <div className="mt-5 grid grid-cols-2 gap-3">
              {photos.map((photo, index) => (
                <div
                  key={photo.id}
                  className="group relative aspect-[4/3] overflow-hidden rounded-lg bg-surface-soft"
                >
                  <Image
                    src={photo.thumbnail_url || photo.url}
                    alt={photo.fileName || `${t.newListing.photos_title} ${index + 1}`}
                    fill
                    className="object-cover"
                    sizes="180px"
                  />
                  {index === 0 && (
                    <span className="absolute left-2 top-2 rounded-md bg-white/92 px-2 py-1 text-[11px] font-bold text-primary shadow-card">
                      {t.newListing.cover_badge}
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => removePhoto(photo.id)}
                    className="absolute right-2 top-2 flex size-8 items-center justify-center rounded-lg bg-white/92 text-destructive opacity-0 shadow-card transition-opacity group-hover:opacity-100"
                    aria-label={t.newListing.photos_remove}
                    title={t.newListing.photos_remove}
                  >
                    <Trash2 className="size-4" />
                  </button>
                </div>
              ))}
            </div>
          )}

          <p className="mt-4 text-meta text-text-muted">
            {t.newListing.photos_count
              .replace("{count}", String(photos.length))
              .replace("{max}", String(maxPhotos))}
          </p>
        </section>

        <section className="rounded-lg border border-border bg-surface-soft p-5">
          <div className="flex items-start gap-3">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-white text-primary shadow-card">
              <Sparkles className="size-4" />
            </div>
            <div className="min-w-0">
              <h3 className="text-card-title text-text-primary">{t.newListing.preview_title}</h3>
              <dl className="mt-4 space-y-3 text-body">
                <PreviewRow label={t.newListing.category_label} value={selectedCategory?.name} />
                <PreviewRow label={t.newListing.city_label} value={cityLabel} />
                <PreviewRow
                  label={t.newListing.condition_label}
                  value={
                    condition === "new" ? t.newListing.condition_new : t.newListing.condition_used
                  }
                />
              </dl>
            </div>
          </div>
        </section>

        <Button
          type="submit"
          disabled={isSubmitDisabled}
          className="h-12 w-full rounded-lg text-[15px] font-bold shadow-[0_12px_28px_rgba(91,69,245,0.24)]"
        >
          {activeMutation.isPending
            ? isEdit
              ? t.newListing.save_loading
              : t.newListing.submit_loading
            : isEdit
              ? t.newListing.save
              : t.newListing.submit}
        </Button>
      </aside>
    </form>
  );
}

function Field({
  label,
  htmlFor,
  required,
  className,
  children,
}: {
  label: string;
  htmlFor?: string;
  required?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  const LabelTag = htmlFor ? "label" : "div";

  return (
    <div className={cn("space-y-2", className)}>
      <LabelTag htmlFor={htmlFor} className="block text-sm font-semibold text-text-primary">
        {label}
        {required && <span className="text-destructive">*</span>}
      </LabelTag>
      {children}
    </div>
  );
}

function SelectField({
  id,
  value,
  onChange,
  disabled,
  placeholder,
  required,
  children,
}: {
  id: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder: string;
  required?: boolean;
  children: React.ReactNode;
}) {
  return (
    <select
      id={id}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      required={required}
      className={cn(
        "h-11 w-full min-w-0 rounded-lg border border-input bg-white px-3 py-2 text-[15px] outline-none transition-colors",
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50",
        "focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50",
      )}
    >
      <option value="">{placeholder}</option>
      {children}
    </select>
  );
}

function PreviewRow({ label, value }: { label: string; value?: string }) {
  return (
    <div className="grid grid-cols-[96px_minmax(0,1fr)] gap-3">
      <dt className="text-text-muted">{label}</dt>
      <dd className="min-w-0 font-semibold text-text-primary">{value || "-"}</dd>
    </div>
  );
}

function defaultCurrencyForLocale(locale: string): string {
  return locale === "ru" ? "RUB" : "USD";
}
