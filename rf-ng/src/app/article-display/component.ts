import { Component, OnInit, OnDestroy, Input, ViewChild, ElementRef } from "@angular/core";
import { ActivatedRoute, Router } from "@angular/router";
import { Location } from '@angular/common';
import { Article, ArticleService } from "../services/article"
import { Observable, Subscription } from "rxjs";
import { Subject } from "rxjs/Subject";
import 'rxjs/add/observable/of'
import 'rxjs/add/operator/distinctUntilChanged'
import 'rxjs/add/operator/map'
import 'rxjs/add/operator/mergeMap'
import 'rxjs/add/operator/switchMap'
import { NgbCarouselConfig, NgbCarousel } from '@ng-bootstrap/ng-bootstrap';

@Component({
    selector: "article-display",
    templateUrl: "./article-display.html",
    styleUrls: ["./article-display.css"],
    providers: [ NgbCarouselConfig ],
    host: {
        '(keydown.arrowUp)': 'keyUp()',
        '(keydown.v)': 'keyView()',
    }
})
export class ArticleDisplayComponent implements OnInit, OnDestroy {
    @Input()
    items: Article[] = []

    slides: Article[] = []

    @ViewChild("carousel")
    private carousel : NgbCarousel;
    @ViewChild("carousel", {read: ElementRef})
    private carouselElement: ElementRef;

    private active: Article
    private offset = new Subject<number>();
    private subscription: Subscription;

    constructor(
        config: NgbCarouselConfig,
        private route: ActivatedRoute,
        private router: Router,
        private location: Location,
        private articleService: ArticleService,
    ) {
        config.interval = 0
        config.wrap = false
        config.keyboard = true
    }

    ngOnInit(): void {
        this.subscription = this.articleService.articleObservable(
        ).switchMap(articles =>
            this.offset.startWith(0).map((offset) : [Article[], number, boolean] => {
                let id = this.route.snapshot.params["articleID"];
                let slides : Article[] = [];
                let index = articles.findIndex(article => article.id == id)

                if (index == -1) {
                    return null
                }

                if (offset != 0) {
                    if (index + offset != -1 && index + offset < articles.length) {
                        index += offset;
                    }

                    let path = this.location.path();
                    path = path.substring(0, path.lastIndexOf("/") + 1) + articles[index].id;
                    this.router.navigateByUrl(path)
                }

                if (index > 0) {
                    slides.push(articles[index-1]);
                }
                slides.push(articles[index]);
                if (index + 1 < articles.length) {
                    slides.push(articles[index + 1]);
                }

                return [slides, articles[index].id, articles[index].read];
            })
        ).filter(
            data => data != null
        ).distinctUntilChanged((a, b) =>
            a[1] == b[1] && a[0].length == b[0].length
        ).flatMap(data => {
            if (data[2]) {
                return Observable.of(data);
            }
            return this.articleService.read(data[1], true).map(s => data);
        }).subscribe(
            data => {
                let [slides, id] = data
                this.carousel.activeId = id.toString();
                this.slides = slides;
                this.active = slides.find(article => article.id == id);

                if (slides.length == 2 && slides[1].id == id) {
                    this.articleService.requestNextPage()
                }
            },
            error => console.log(error)
        );

        this.carouselElement.nativeElement.focus()
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    slideEvent(next: boolean) {
        if (next) {
            this.offset.next(1);
        } else {
            this.offset.next(-1);
        }
    }

    favor(id: number, favor: boolean) {
        this.articleService.favor(id, favor).subscribe(
            success => { },
            error => console.log(error)
        )
    }

    keyUp() {
        let path = this.location.path();
        this.router.navigateByUrl(path.substring( 0, path.indexOf("/article/")))
    }

    keyView() {
        if (this.active != null) {
            window.open(this.active.link, "_blank");
        }
    }
}