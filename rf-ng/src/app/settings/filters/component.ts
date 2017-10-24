import { Component, OnInit, Inject, ViewChild, ElementRef } from "@angular/core" ;
import { FormBuilder, FormGroup, Validators, ValidationErrors } from '@angular/forms';
import { MatDialog, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';
import { UserService, User } from "../../services/user";
import { FeedService, Feed } from "../../services/feed";
import { TagService, Tag } from "../../services/tag";
import { FaviconService } from "../../services/favicon";
import { Observable } from "rxjs";
import 'rxjs/add/operator/combineLatest'

interface Filter {
    term: string,
    inverse?: boolean,
    matchURL?: boolean,
    matchTitle?: boolean,
    tagID?: number,
    feedIDs?: number[],
    inverseFeeds?: boolean,
}

@Component({
    selector: "settings-filters",
    templateUrl: "./filters.html",
    styleUrls: ["../common.css", "./filters.css"]
})
export class FiltersSettingsComponent implements OnInit {
    filters = new Array<Filter>()

    feeds: Feed[];
    tags: Tag[];

    constructor(
        private userService: UserService,
        private feedService: FeedService,
        private tagService: TagService,
        private dialog: MatDialog,
    ) {
    }

    ngOnInit(): void {
        this.feedService.getFeeds().combineLatest(
            this.tagService.getTags(),
            this.userService.getCurrentUser(),
            (feeds, tags, user) : [Feed[], Tag[], User] => [feeds, tags, user]
        ).subscribe(
            data => {
                this.feeds = data[0];
                this.tags = data[1];
                this.filters = data[2].profileData["filters"] || [];
            },
            error => console.log(error),
        )
    }

    addFilter() {
        this.dialog.open(NewFilterDialog, {
            width: "350px",
            data: {feeds: this.feeds, tags: this.tags},
        }).afterClosed().subscribe(
            v => this.ngOnInit(),
        );
    }

    feedsLabel(ids: number[]) : string {
        let filtered = this.feeds.filter(feed => ids.indexOf(feed.id) != -1).map(filter => filter.title);

        return filtered.length ? filtered.join(", ") : `${ids}`;
    }

    tagLabel(id: number) : string {
        let filtered = this.tags.filter(tag => tag.id == id).map(tag => tag.value);

        return filtered.length ? filtered[0] : `${id}`;
    }

    deleteFilter(event: Event, filter: Filter) {
        this.userService.getCurrentUser().flatMap(user => {
            let profile = user.profileData || new Map<string, any>();
            let filters = (profile["filters"] || []) as Filter[];

            let filtered = filters.filter(f =>
                f.term != filter.term ||
                f.inverse != filter.inverse ||
                f.matchURL != filter.matchURL ||
                f.matchTitle != filter.matchTitle ||
                f.tagID != filter.tagID ||
                f.feedIDs != filter.feedIDs ||
                f.inverseFeeds != filter.inverseFeeds
            )

            if (filtered.length == filters.length) {
                return Observable.of(true);
            }

            profile["filters"] = filtered;

            return this.userService.setProfile(user.login, profile);
        }).subscribe(
            success => {
                if (success) {
                    let el = event.target["parentNode"];
                    while ((el = el.parentElement) && !el.classList.contains("filter"));
                    el.parentNode.removeChild(el);
                }
            },
            error => console.log(error),
        );
    }
}

@Component({
    selector: 'new-filter-dialog',
    templateUrl: 'new-filter-dialog.html',
    styleUrls: ["../common.css", "./filters.css"]
})
export class NewFilterDialog {
    form: FormGroup

    feeds: Feed[];
    tags: Tag[];

    constructor(
        private dialogRef: MatDialogRef<NewFilterDialog>,
        private userService: UserService,
        @Inject(MAT_DIALOG_DATA) private data: {feeds: Feed[], tags: Tag[]},
        formBuilder: FormBuilder,
    ) {
        this.form = formBuilder.group({
            term: ['', Validators.required],
            inverse: [false],
            matchURL: [true],
            matchTitle: [false],
            useFeeds: [true],
            feeds: [[] as number[]],
            tag: [],
            inverseFeeds: [false],
        }, {
            validator: (control: FormGroup) : ValidationErrors | null => {
                if (
                    !control.controls.matchURL.value &&
                    !control.controls.matchTitle.value) {
                    return {"nomatch": true};
                }
                return null;
            },
        })

        this.feeds = data.feeds;
        this.tags = data.tags;
    }

    save() {
        if (!this.form.valid) {
            return;
        }

        let value = this.form.value;

        this.userService.getCurrentUser().flatMap(user => {
            let profile = user.profileData || new Map<string, any>();
            let filters = (profile["filters"] || []) as Filter[];
            let filter : Filter = {
                term: value.term,
            }

            filter.inverse = value.inverse;
            filter.matchTitle = value.matchTitle;
            filter.matchURL = value.matchURL;
            filter.inverseFeeds = value.inverseFeeds;

            if (value.useFeeds) {
                if (value.feeds && value.feeds.length > 0) {
                    filter.feedIDs = value.feeds;
                }
            } else {
                if (value.tag) {
                    filter.tagID = value.tag;
                }
            }

            filters.push(filter);

            profile["filters"] = filters;

            return this.userService.setProfile(user.login, profile);
        }).subscribe(
            success => this.close(),
            error => console.log(error),
        )
    }

    close() {
        this.dialogRef.close();
    }
}